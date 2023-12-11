package dlab

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"path/filepath"
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
	l, e := net.Listen("tcp", string(n.address))

	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)

}

func (n *Node) ServerSecure() {
	rpc.Register(n)
	rpc.HandleHTTP()

	cert, err := tls.LoadX509KeyPair("server.pem", "server-key.pem")
	if err != nil {
		log.Fatal("Error loading server certificates:", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		// Other TLS configuration options as needed
	}

	l, e := tls.Listen("tcp", string(n.address), tlsConfig)
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
	fmt.Println("filepath:", filepath)
	wd, _ := os.Getwd()
	fmt.Println("working directory:", wd)
	println(nodeAdressToGiveFile) //temp to use var
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
	pubKey, err := node.KeyRequest(nodeAdressToGiveFile)
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
	err = call(string(nodeAdressToGiveFile), "Node.StoreFileRPC", &args, &reply)
	if err != nil || reply.Error != nil {
		fmt.Println(err, reply.Error)
		return errors.New("something went wrong when calling storefilerpc")
	}
	//

	//Send file securely to found address

	/*
		fmt.Println("Found file at node adress: ", nodeAdressWhoHasFile)
		FilePath := filepath.Join("nodeFiles", node.Name, "upload", filename) //nodeFiles/node.Name/upload/filename
		file, err := os.Open(FilePath)
		if err != nil {
			fmt.Println("Cant open file!")
			return err
		}
		defer file.Close()

		fileInfo := File{}
		fileInfo.Name = filename
		fileInfo.Id = hashString(filename)
		fileInfo.Id.Mod(fileInfo.Id, hashMod)
		fileInfo.Content, err = io.ReadAll(file)
		if err != nil {
			fmt.Println("Cant read fileconent!")
			return err
		}
		fileInfo.Content, err = node.encryptFileContent(fileInfo.Content)
		if err != nil {
			return err
		}
		reply := StoreFileReply{}
		err = call(string(nodeAdressWhoHasFile), "Node.StoreFileRPC", fileInfo, &reply)
		if err != nil || reply.Error != nil {
			fmt.Println(err, reply.Error)
			return errors.New("something went wrong when calling storefilerpc")
		}*/
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

func GetFile(fileName string, node *Node) error {
	_, nodeAdressWhoHasFile, err := Lookup(fileName, node)
	if err != nil {
		fmt.Println("Couldnt complete Lookup ")
		return err
	}
	fmt.Println("Found file at node adress: ", nodeAdressWhoHasFile)

	fileInfo := File{}
	fileInfo.Name = fileName
	fileInfo.Id = hashString(fileName)
	fileInfo.Id.Mod(fileInfo.Id, hashMod)

	err = call(string(nodeAdressWhoHasFile), "GetFileRPC", fileInfo, &fileInfo)
	if err != nil {
		fmt.Println("couldnt get file")
		return err
	}
	fileInfo.Content, err = node.decryptFileContent(fileInfo.Content)
	if err != nil {
		fmt.Println("couldnt decrypt file")
		return err
	}

	err = node.StoreFileOnNode(fileInfo)
	if err != nil {
		fmt.Println("couldnt save file on node")
		return err
	}
	return nil
}

// store file at node
func (node *Node) StoreFileOnNode(fileInfo File) error {
	fileInfo.Id.Mod(fileInfo.Id, hashMod)
	saveFilePath := filepath.Join("nodeFiles", node.Name, "download", fileInfo.Name)

	file, err := os.Create(saveFilePath)
	if err != nil {
		fmt.Println("Couldnt create file ", err)
		return err
	}

	defer file.Close()

	_, err = file.Write(fileInfo.Content)
	if err != nil {
		fmt.Println("Couldnt write content to file ", err)
		return err
	}

	return nil
}
