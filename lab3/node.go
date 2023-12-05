package main

import (
	"fmt"
	"math/big"
	"strconv"
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
	Id *big.Int // Default hash sum of id, can be overrided with -a

	//Chord:
	successors  []NodeAddress // List of addresses (IP & Port) to the next several nodes on the ring
	predecessor NodeAddress   // An address to the previous node on the circle
	fingerTable []NodeAddress // A list of addresses of nodes on the circle
	//TODO: encryption
	address NodeAddress // address is both IP & Port
	Bucket  map[*big.Int]string
}

func (n *Node) findSuccessor(id string) {

}

func NewNode(args *Arguments) *Node {
	node := &Node{}
	var nodeName string
	//TODO: Check if ip = localhost
	node.address = NodeAddress(fmt.Sprintf("IP: %s, Port: %d", args.IpAdress, args.Port))
	if args.Identifier == "NodeAdress" {
		nodeName = string(node.address)
	} else {
		nodeName = args.Identifier
	}
	node.Id = hashString(string(nodeName))
	node.Id.Mod(node.Id, hashMod)
	node.fingerTable = make([]NodeAddress, fingerTableSize+1) // finger table entry (using 1-based numbering).
	node.successors = make([]NodeAddress, args.NumSuccessors) //set size of successors array
	node.predecessor = ""
	node.Bucket = make(map[*big.Int]string)
	node.initSuccessors()
	node.initFingerTable()

	//TODO: create folders for node

	return node
}

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

func (n *Node) join(args *Arguments) {

	n.findSuccessor(args.JoinIpAdress + ":" + strconv.FormatInt(args.JoinPort, 10))

}

// stabilize: This function is periodically used to check the immediate successor and notify it about this node
//func stabilize(node Node) {
//Ask your current successor who their predecessor is, and get their successor table
//if you are not the predecessor anymore notify this new successor and update your predeccessor
//if your succesor doesn't reply truncate it from your list, if the list then is empty make yourself successor
/*	args := PredecessorCall{}
	reply := PredecessorResponse{}
	err := call(string(node.successors[0]), "functions.Predecessor", &args, &reply)
	if reply == node.address {
	}

	//TODO

}*/

// Notify: This function is used when another node says it might be the predecessor of this node
func (n *Node) Notify(np string) {

}

// fixFingers: Periodically called to update/confirm part of the finger table (not all entries at once)
func fixFingers() {
	//keep a global variable of "next entry to check" and increment it as this is periodically run
}

// checkPredecessor: This function should occasionally be called to check if the current predecessor is alive
func (n *Node) checkPredecessor() {
	args := PingArgs{}
	reply := PingReply{}
	err := call(string(n.predecessor), "functions.Ping", &args, &reply) // "functions.Ping" might need to be changed
	if err != nil {
		n.predecessor = ""
	}
}
