package dlab

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strings"
)

func hashString(elt string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(elt))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}

// start a thread that listens for RPCs from other nodes
func (n *Node) Server() {
	rpc.Register(n)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", string(n.myInfo.Address))

	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)

}

func getLocalAddress() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

func between(start *big.Int, elt *big.Int, end *big.Int, inclusive bool) bool {

	if end.Cmp(start) > 0 {
		return (start.Cmp(elt) < 0 && elt.Cmp(end) < 0) || (inclusive && elt.Cmp(end) == 0)
	} else {
		return start.Cmp(elt) < 0 || elt.Cmp(end) < 0 || (inclusive && elt.Cmp(end) == 0)
	}
}

// Stores a file from another node at this node
func FindAndstoreFile(filepath string, node *Node) error {
	filepath = strings.TrimSpace(filepath)
	index := strings.LastIndex(filepath, "/")
	if index == -1 {
		return errors.New("incorrect filepath")
	}
	filename := filepath[index+1:]

	key, nodeAdressToGiveFile, err := Lookup(filename, node)
	if err != nil {
		return err
	}

	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("Cant open file!")
		return err
	}
	defer file.Close()

	fileInfo := File{}
	fileInfo.Name = filename
	fileInfo.Id = key
	fileInfo.Content, err = io.ReadAll(file)
	if err != nil {
		fmt.Println("Cant read fileconent!")
		return err
	}

	//Encryption
	pubKey, err := node.KeyRequest(nodeAdressToGiveFile.Address)
	if err != nil {
		fmt.Println("couldnt get rsa public key!")
		return err
	}
	fileInfo.Content, err = node.encryptFileContent(fileInfo.Content, pubKey)
	if err != nil {
		return err
	}

	args := StoreFileArgs{fileInfo}
	reply := StoreFileReply{}
	err = call(string(nodeAdressToGiveFile.Address), "Node.StoreFileRPC", &args, &reply)
	if err != nil || reply.Error != nil {
		fmt.Println(err, reply.Error)
		return errors.New("something went wrong when calling storefilerpc")
	}

	return nil
}

// encrypt content of a file
func (node *Node) encryptFileContent(content []byte, pubKey *rsa.PublicKey) ([]byte, error) {
	encryptedContent, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, content)
	if err != nil {
		fmt.Println("couldnt encrypt file", err)
		return make([]byte, 0), err
	}
	return encryptedContent, nil
}

// decrypt content of a file
func (node *Node) decryptFileContent(content []byte) ([]byte, error) {
	decryptedContent, err := rsa.DecryptPKCS1v15(rand.Reader, node.PrivateKey, content)
	if err != nil {
		fmt.Println("couldnt decrypt file", err)
		return make([]byte, 0), err
	}
	return decryptedContent, nil
}
