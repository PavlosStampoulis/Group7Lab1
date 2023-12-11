package dlab

import (
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"log"
	"math/big"
	"net/rpc"
	"net/rpc/jsonrpc"
)

func call(address string, method string, args interface{}, reply interface{}) error {
	c, err := rpc.DialHTTP("tcp", address)
	//sockname := coordinatorSock()
	//c, err := rpc.DialHTTP("unix", sockname)
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

func callSecure(address string, method string, args interface{}, reply interface{}) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // You may want to set this to false in a production environment
	}

	conn, err := tls.Dial("tcp", address, tlsConfig)
	if err != nil {
		log.Println("dialing:", err)
		return err
	}
	defer conn.Close()

	client := rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn))

	err = client.Call(method, args, reply)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func (n *Node) Ping(args *PingArgs, reply *PongReply) error {

	return nil
}

/*
func (n *Node) getNumberSuccessors(args *NumberSuccessorsCall, reply *NumberSuccessorsResponse) error {

	reply.numberofsuccessors = globalNumberSuccessors

	return nil
}*/

// NotifyReceiver: recieve notification from node believing to be our Predecessor
func (n *Node) NotifyReceiver(args *NotifyArgs, reply *NotifyReply) error {
	pre := hashString(string(n.predecessor))
	pre.Mod(pre, hashMod)
	bet := hashString(string(args.Address))
	bet.Mod(bet, hashMod)
	if (n.predecessor == "") || between(pre, bet, n.Id, true) {
		fmt.Println("should set predecessor, ", args.Address)
		n.predecessor = args.Address
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

	prev := hashString(string(n.address))
	prev.Mod(prev, hashMod)
	for i, node := range n.successors {
		curr := hashString(string(node))
		curr.Mod(curr, hashMod)
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
