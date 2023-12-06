package dlab

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
)

var m = 6
var fingerTableSize = 6                                                  // = m
var hashMod = new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(m)), nil) // 2^m

type Key string
type NodeAddress string

/*type fingerEntry struct {
	ChordRingAdress [] byte
	Adress 	NodeAddress		adress of node or file itself
}*/

type Node struct {
	Id   *big.Int // Default hash sum of id, can be overrided with -a
	next int      // pointer to next fingertable entry

	//Chord:
	successors  []NodeAddress // List of addresses (IP & Port) to the next several nodes on the ring
	predecessor NodeAddress   // An address to the previous node on the circle
	fingerTable []NodeAddress // A list of addresses of nodes on the circle
	address     NodeAddress   // address is both IP & Port
	Bucket      map[*big.Int]string

	//encryption
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

/*func (n *Node) findSuccessor(id string) {

}*/

func NewNode(args *Arguments) *Node {
	node := &Node{}
	var nodeName string

	if args.IpAdress == "localhost" || args.IpAdress == "0.0.0.0" {
		args.IpAdress = getLocalAddress()
	}

	node.address = NodeAddress(fmt.Sprintf("%s:%d", args.IpAdress, args.Port))
	if args.Identifier == "NodeAdress" {
		nodeName = string(node.address)
	} else if args.Identifier != "" {
		nodeName = args.Identifier
	} else { // Create an id: "192.168.1.1:8080" becomes "192168118080"
		// Remove dots and colon
		temp := strings.Replace(string(node.address), ".", "", -1)
		nodeName = strings.Replace(temp, ":", "", -1)
	}
	node.Id = hashString(string(nodeName))
	node.Id.Mod(node.Id, hashMod)
	node.fingerTable = make([]NodeAddress, fingerTableSize+1) // finger table entry (using 1-based numbering).
	node.successors = make([]NodeAddress, args.NumSuccessors) //set size of successors array
	node.predecessor = ""
	node.Bucket = make(map[*big.Int]string)
	node.initSuccessors()
	node.initFingerTable()
	node.next = 0

	node.createNodeFolder(nodeName)
	fmt.Println("Node folders has been configured")

	if err := node.initBucket(nodeName); err != nil {
		fmt.Println("cant read directory chordStorage", err)
	}

	if err := node.encrypt(nodeName); err != nil {
		fmt.Println("cant encrypt node!", err)
	}

	return node
}

func (node *Node) encrypt(nodeName string) error {
	// Generate and set RSA private key
	privateKeyPath := filepath.Join("nodeFiles", nodeName, "privateKey.pem")
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		// Private key doesn't exist, generate and save it
		if err := node.createRSAPrivateKey(nodeName); err != nil {
			fmt.Errorf("failed to create RSA private key: %v", err)
			return err
		}
	}

	privateHandler, err := os.Open(privateKeyPath)
	if err != nil {
		fmt.Errorf("failed to open private key file: %v", err)
		return err
	}
	defer privateHandler.Close()

	privateKeyBuffer, err := io.ReadAll(privateHandler)
	if err != nil {
		fmt.Errorf("failed to read private key file: %v", err)
		return err
	}

	priBlock, _ := pem.Decode(privateKeyBuffer)
	privateKey, err := x509.ParsePKCS1PrivateKey(priBlock.Bytes)
	if err != nil {
		fmt.Errorf("failed to parse private key: %v", err)
		return err
	}

	node.PrivateKey = privateKey
	node.PublicKey = &node.PrivateKey.PublicKey

	return nil
}

func (node *Node) createRSAPrivateKey(nodeName string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key pair: %v", err)
	}

	node.PrivateKey = privateKey
	node.PublicKey = &privateKey.PublicKey

	// Encode private key to PEM format
	privatePEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// Save private key to file
	privateKeyPath := filepath.Join("nodeFiles", nodeName, "privateKey.pem")
	privateFile, err := os.Create(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %v", err)
	}
	defer privateFile.Close()

	err = pem.Encode(privateFile, privatePEM)
	if err != nil {
		return fmt.Errorf("failed to encode private key to PEM: %v", err)
	}

	return nil
}

func (node *Node) initBucket(nodeName string) error {
	directoryChordStorage, err := os.ReadDir(filepath.Join("nodeFiles", nodeName, "chordStorage"))
	if err != nil {
		return err
	}
	for _, file := range directoryChordStorage {
		fileName := file.Name()
		hashedFileName := hashString(fileName)
		hashedFileName.Mod(hashedFileName, hashMod)
		node.Bucket[hashedFileName] = fileName
	}
	return nil
}

func createDirectory(path string) error {
	if !directoryExists(path) {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return fmt.Errorf("could not create folder %s: %v", path, err)
		}
	}
	return nil
}
func directoryExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

/*
	 Tree structure:
	 Lab3/
	 └── nodeFiles/

		├── nodename1/
		│   ├── upload
		│   ├── download
		│   └── chordStorage
		├── nodename2/
		│   ├── upload
		│   ├── download
		│   └── chordStorage
		└── nodename3/
		    ├── upload
		    ├── download
		    └── chordStorage

	 Creates dirs for nodes
*/
func (node *Node) createNodeFolder(nodeName string) {
	fmt.Println("\n Setting up node folders...")
	basePath := filepath.Join("nodeFiles", nodeName)

	if err := createDirectory(basePath); err != nil { //Create
		fmt.Println(err)
		return
	}

	subfolders := []string{"upload", "download", "chordStorage"}
	for _, folder := range subfolders {
		folderPath := filepath.Join(basePath, folder)
		if err := createDirectory(folderPath); err != nil {
			fmt.Println(err)
			return
		}
	}

}

/*
Tree structure:
 Lab3/
 └── nodeFiles/

	├── nodename1/
	│   ├── upload
	│   ├── download
	│   └── chordStorage
	├── nodename2/
	│   ├── upload
	│   ├── download
	│   └── chordStorage
	└── nodename3/
	    ├── upload
	    ├── download
	    └── chordStorage
*/

func (node *Node) initSuccessors() {
	for i := 0; i < len(node.successors); i++ {
		node.successors[i] = ""
	}
}

func (node *Node) initFingerTable() {
	//TODO: add chordringadresses
	node.fingerTable[0] = node.address
	for i := 1; i < fingerTableSize+1; i++ {
		node.fingerTable[i] = node.address
	}
}

// Make predecessor empty and point all successors to self
func (node *Node) CreateChord() {
	node.predecessor = NodeAddress("")
	for i := 0; i < len(node.successors); i++ {
		node.successors[i] = node.address
	}
	//fingertable set itself
	//n.fingerTable[0] = NodeAddress(n.address)

	//successor equal to itself
	//n.successors[0] = NodeAddress(n.address)

}

func (n *Node) Join(args *Arguments) error {
	//n.findSuccessor(args.JoinIpAdress + ":" + strconv.FormatInt(args.JoinPort, 10))
	return nil
}

// stabilize: This function is periodically used to check the immediate successor and notify it about this node
func stabilize(node Node) {
	//Ask your current successor who their predecessor is, and get their successor table
	//if you are not the predecessor anymore notify this new successor and update your successer
	//if your succesor doesn't reply truncate it from your list, if the list then is empty make yourself successor
	args := StabilizeCall{}
	reply := StabilizeResponse{}
	err := call(string(node.successors[0]), "Node.StabilizeData", &args, &reply)
	if err != nil {
		if len(node.successors) > 0 {
			node.successors = node.successors[1:]
		} else {
			node.successors[0] = node.address
		}
	}

	copy := []NodeAddress{}
	if reply.predecessor == node.address {
		if len(reply.successors_successors) == int(reply.numberofsuccessors) {
			copy = reply.successors_successors[:len(reply.successors_successors)-1]
			oneelement := []NodeAddress{reply.address}
			copy = append(oneelement, copy...)
		} else {
			oneelement := []NodeAddress{reply.address}
			copy = append(oneelement, reply.successors_successors...)
		}
	} else {
		if len(reply.successors_successors) == (int(reply.numberofsuccessors)) {
			copy = reply.successors_successors[:len(reply.successors_successors)-2]
			oneelement := []NodeAddress{reply.predecessor, reply.address}
			copy = append(oneelement, copy...)
		} else if len(reply.successors_successors) == (int(reply.numberofsuccessors) - 1) {
			copy = reply.successors_successors[:len(reply.successors_successors)-1]
			oneelement := []NodeAddress{reply.predecessor, reply.address}
			copy = append(oneelement, reply.successors_successors...)
		} else {
			oneelement := []NodeAddress{reply.predecessor, reply.address}
			copy = append(oneelement, reply.successors_successors...)
		}
	}

	node.successors = copy

	node.Notify(node.successors[0])

}
func removeElement(slice []NodeAddress, index int) []NodeAddress {
	return append(slice[:index], slice[index+1:]...)
}

// Notify: notify a node to tell we think we're their predecessor
func (n *Node) Notify(np NodeAddress) {
	args := NotifyArgs{}
	reply := NotifyReply{}
	err := call(string(np), "Node.NotifyReceiver", &args, &reply)
	if err != nil {
		//Handle error
		fmt.Println("Node either left or went down")
	}

}

func findSuccessor() NodeAddress {

	finger := NodeAddress("") //TODO implement here or in fix fingers, currently placeholder
	return finger
}

// fixFingers: Periodically called to update/confirm part of the finger table (not all entries at once)
func (n *Node) fixFingers() {
	n.fingerTable[n.next] = findSuccessor() //TODO placeholder
	//keep a global variable of "next entry to check" and increment it as this is periodically run
	n.next = (n.next + 1) % len(n.fingerTable)

}

// checkPredecessor: This function should occasionally be called to check if the current predecessor is alive
func (n *Node) checkPredecessor() {
	args := PingArgs{}
	reply := PingReply{}
	err := call(string(n.predecessor), "Node.Ping", &args, &reply)
	if err != nil {
		n.predecessor = ""
	}
}
