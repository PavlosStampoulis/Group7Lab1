package dlab

import (
	"crypto/rsa"
	"fmt"
	"log"
	"math/big"
	"net/rpc"
)

func call(address string, method string, args interface{}, reply interface{}) error {
	c, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		log.Println("dialing:", err)
		return err
	}
	defer c.Close()

	err = c.Call(method, args, reply)
	if err == nil {
		return nil
	}

	fmt.Println(err)
	return err
}

func (n *Node) Ping(args *PingArgs, reply *PongReply) error {

	return nil
}

// NotifyReceiver: recieve notification from node believing to be our Predecessor
func (n *Node) NotifyReceiver(args *NotifyArgs, reply *NotifyReply) error {
	if (string(n.predecessor.Address) == "") || between(n.predecessor.Id, args.Info.Id, n.myInfo.Id, true) {
		n.predecessor = args.Info
		reply.Ok = true
		return nil
	}
	reply.Ok = false
	return nil
}

func (n *Node) StabilizeData(args *StabilizeCall, reply *StabilizeResponse) error {
	reply.Predecessor = n.predecessor
	reply.Successors_successors = n.successors

	return nil
}

func (n *Node) FindSuccessor(args *FindSuccessorArgs, reply *FindSuccessorReply) error {

	prev := n.myInfo.Id
	prev.Mod(prev, hashMod)
	for i, node := range n.successors {
		curr := node.Id
		if between(prev, args.Id, curr, true) {
			reply.Found = true
			reply.RetAddress = n.successors[i]
			return nil
		}
		prev = curr
	}
	reply.Found = false
	reply.RetAddress = n.closestPredecessor(args.Id)
	return nil
}

// inserts key, vals into active node
// Moves nodes data before a node shut down
func (n *Node) MoveAll(bucket map[*big.Int]string, emptyReply *struct{}) error {
	for key, val := range bucket {
		n.Bucket[key] = val
	}
	return nil
}

/*func (n *Node) DoMoveAll(adress string, empty *struct{}) error {
	tempBucket := make(map[*big.Int]string)
	for key, val := range n.Bucket {
		if between((n.predecessor[0]), (key) {

		}
	}

	call(adress, "MoveAll")
	return nil
}*/

func (n *Node) KeyRequest(node NodeAddress) (*rsa.PublicKey, error) {
	args := RequestPubKeyArgs{}
	reply := RequestPubKeyReply{}
	err := call(string(node), "Node.RecieveKeyRequest", &args, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Key, nil
}

func (n *Node) RecieveKeyRequest(args *RequestPubKeyArgs, reply *RequestPubKeyReply) error {
	reply.Key = n.PublicKey
	return nil
}
