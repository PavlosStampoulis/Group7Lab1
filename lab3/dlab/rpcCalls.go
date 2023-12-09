package dlab

import (
	"fmt"
	"log"
	"math/big"
	"net/rpc"
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
