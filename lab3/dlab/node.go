package dlab

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var m = 6
var fingerTableSize = 6                                                  // = m
var hashMod = new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(m)), nil) // 2^m
var maxSteps = 35

type Key string
type NodeAddress string

/*type fingerEntry struct {
	ChordRingAdress [] byte
	Adress 	NodeAddress		adress of node or file itself
}*/

type Node struct {
	Id   *big.Int // Default hash sum of id, can be overrided with -a
	next int      // pointer to next fingertable entry
	Name string

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

	if args.IpAdress == "localhost" || args.IpAdress == "0.0.0.0" {
		args.IpAdress = getLocalAddress()
	}

	node.address = NodeAddress(fmt.Sprintf("%s:%d", args.IpAdress, args.Port))
	if args.Identifier == "NodeAdress" {
		node.Name = string(node.address)
	} else if args.Identifier != "" {
		node.Name = args.Identifier
	} else { // Create an id: "192.168.1.1:8080" becomes "192168118080"
		// Remove dots and colon
		temp := strings.Replace(string(node.address), ".", "", -1)
		node.Name = strings.Replace(temp, ":", "", -1)
	}
	node.Id = hashString(string(node.Name))
	node.Id.Mod(node.Id, hashMod)
	node.fingerTable = make([]NodeAddress, fingerTableSize+1) // finger table entry (using 1-based numbering).
	node.successors = make([]NodeAddress, args.NumSuccessors) //set size of successors array
	node.predecessor = ""
	node.Bucket = make(map[*big.Int]string)
	node.initSuccessors()
	node.initFingerTable()
	node.next = 0

	node.createNodeFolder()
	fmt.Println("Node folders has been configured")

	if err := node.initBucket(); err != nil {
		fmt.Println("cant read directory chordStorage", err)
	}

	if err := node.encrypt(); err != nil {
		fmt.Println("cant encrypt node!", err)
	}

	return node
}

func (node *Node) encrypt() error {
	// Generate and set RSA private key
	privateKeyPath := filepath.Join("nodeFiles", node.Name, "privateKey.pem")
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		// Private key doesn't exist, generate and save it
		if err := node.createRSAPrivateKey(); err != nil {
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

func (node *Node) createRSAPrivateKey() error {
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
	privateKeyPath := filepath.Join("nodeFiles", node.Name, "privateKey.pem")
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

func (node *Node) initBucket() error {
	directoryChordStorage, err := os.ReadDir(filepath.Join("nodeFiles", node.Name, "chordStorage"))
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
func (node *Node) createNodeFolder() {
	fmt.Println("\n Setting up node folders...")
	basePath := filepath.Join("nodeFiles", node.Name)

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
		node.successors[i] = node.address
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
	node.initSuccessors()
	//node.initFingerTable()
	//go node.timedCalls()
	//fingertable set itself
	//n.fingerTable[0] = NodeAddress(n.address)

	//successor equal to itself
	//n.successors[0] = NodeAddress(n.address)

}

func (n *Node) Join(args *Arguments) error {
	askN := NodeAddress(args.JoinIpAdress + ":" + strconv.FormatInt(args.JoinPort, 10))
	joinN, err := find(n.Id, askN)
	if err != nil {
		fmt.Println("Failed to find place for " + n.address)
	}
	fmt.Println("Told to join " + joinN)
	n.successors[0] = joinN
	n.Notify(joinN)
	//go n.timedCalls()
	//n.findSuccessor(args.JoinIpAdress + ":" + strconv.FormatInt(args.JoinPort, 10))
	return nil
}

func (n *Node) TimedCalls() {
	for {
		n.stabilize()
		time.Sleep(300 * time.Millisecond)
		n.fixFingers()
		time.Sleep(300 * time.Millisecond)
		n.checkPredecessor()
		time.Sleep(300 * time.Millisecond)
	}
}

// stabilize: This function is periodically used to check the immediate successor and notify it about this node
func (n *Node) stabilize() {
	if n.successors[0] == n.address {
		if n.predecessor != "" {
			n.successors[0] = n.predecessor
			n.Notify(n.successors[0])
		}
		return
	}

	//Ask your current successor who their predecessor is, and get their successor table
	//if you are not the predecessor anymore notify this new successor and update your successer
	//if your succesor doesn't reply truncate it from your list, if the list then is empty make yourself successor
	args := StabilizeCall{}
	reply := StabilizeResponse{}
	err := call(string(n.successors[0]), "Node.StabilizeData", &args, &reply)
	if err != nil {
		n.successors = append(n.successors[1:], n.address)
	} else if reply.Predecessor == n.address {
		if cap(n.successors) <= len(reply.Successors_successors) {
			n.successors = append(n.successors[:1], reply.Successors_successors[:cap(n.successors)-1]...)
		} else {
			n.successors = append(n.successors[:1], reply.Successors_successors...)
		}
	} else {
		n.successors[1] = n.successors[0]
		n.successors[0] = reply.Predecessor
		if len(reply.Successors_successors) <= cap(reply.Successors_successors)-1 {
			n.successors = append(n.successors[:2], reply.Successors_successors[:cap(n.successors)-2]...)
		} else {
			n.successors = append(n.successors[:2], reply.Successors_successors...)
		}
	}
	n.fingerTable[0] = n.successors[0]
	n.Notify(n.successors[0])

}

// Notify: notify a node to tell we think we're their predecessor
func (n *Node) Notify(np NodeAddress) {
	args := NotifyArgs{n.address}
	reply := NotifyReply{}
	err := call(string(np), "Node.NotifyReceiver", &args, &reply)
	if err != nil {
		//Handle error
		fmt.Println("Node either left or went down")
	}

}

// fixFingers: Periodically called to update/confirm part of the finger table (not all entries at once)
func (n *Node) fixFingers() {

	exp := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(n.next-1)), nil)
	sum := new(big.Int).Add(n.Id, exp)
	fingerID := new(big.Int).Mod(sum, hashMod)
	node, err := find(fingerID, n.successors[0])
	if err != nil {
		fmt.Println("Error finger searching for " + fingerID.String())
	}
	n.fingerTable[n.next] = node

	//n.fingerTable[n.next] = findSuccessor() //TODO placeholder Klar ovanför tror jag
	//keep a global variable of "next entry to check" and increment it as this is periodically run
	n.next = (n.next + 1) % len(n.fingerTable)

}

// checkPredecessor: This function should occasionally be called to check if the current predecessor is alive
func (n *Node) checkPredecessor() {
	if n.predecessor == "" {
		return
	}
	args := PingArgs{}
	reply := PongReply{}
	err := call(string(n.predecessor), "Node.Ping", &args, &reply)
	if err != nil {
		n.predecessor = ""
	}
}

// closestPredecessor: check your finger table to find the node closest before id
// if finger table not initiated, return yourself and don't run the rest
func (n *Node) closestPredecessor(id *big.Int) NodeAddress {
	if n.fingerTable[0] == n.address {
		return n.successors[0]
	}
	closestPre := n.address
	for _, node := range n.fingerTable {
		nodeHash := hashString(string(node))
		x := nodeHash.Cmp(id)
		if x == -1 {
			closestPre = node
		} else if x == 0 {
			return closestPre
		}
	}
	return closestPre
}

// find succesor of id, iterative local version (makes use of RPC version)
func find(id *big.Int, start NodeAddress) (NodeAddress, error) {
	found, nextNode := false, start
	var err error
	for i := 0; i < maxSteps; i++ {
		args := FindSuccessorArgs{id}
		reply := FindSuccessorReply{}
		err = call(string(nextNode), "Node.FindSuccessor", &args, &reply)
		if err != nil {
			fmt.Println("Error find searching for " + id.String())
		}
		found = reply.Found
		if found {
			return reply.RetAddress, err
		}

		if nextNode == reply.RetAddress { //no more successors to check through
			return reply.RetAddress, nil
		}
		nextNode = reply.RetAddress
	}
	return NodeAddress(""), err
}

// Return nodeadress to successor who has the key			key is a filename
func Lookup(key string, n *Node) (NodeAddress, error) {
	hashedKey := hashString(key)
	nodeAdress, err := find(hashedKey, n.address)
	if err != nil {
		return "", err
	} else {
		return nodeAdress, nil
	}

}
func (node *Node) StoreFileRPC(file File, reply *StoreFileReply) error {

	reply.Succeeded = node.StoreFileInChordStorage(file)
	if !reply.Succeeded {
		err := errors.New("failed to store file")
		reply.Error = err
		return err
	}
	reply.Error = nil
	fmt.Println("File has been successfully stored")
	return nil
}

// Save the file in the nodes chordStorage
func (node *Node) StoreFileInChordStorage(file File) bool {

	file.Id.Mod(file.Id, hashMod) // convert name back to normal
	filePath := filepath.Join("nodeFiles", node.Name, "chordStorage", file.Name)

	fileNew, err := os.Create(filePath)
	if err != nil {
		fmt.Println("couldnt create file")
		return false
	}
	defer fileNew.Close()

	_, err = fileNew.Write(file.Content)
	if err != nil {
		fmt.Println("couldnt write content to file!")
		return false
	}

	return true
}
