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
	if (n.predecessor == "") || between(n.predecessor, args.Address, n.Id, true) {
		fmt.Println("should set predecessor, ", args.Address)
		n.predecessor = args.Address
		reply.Ok = true
		return nil
	}

	reply.Ok = false
	return nil
}

func (n *Node) StabilizeData(args *StabilizeCall, reply *StabilizeResponse) error {

	reply.Address = n.address
	reply.Predecessor = n.predecessor
	reply.Successors_successors = n.successors

	return nil
}

func (n *Node) FindSuccessor(args *FindSuccessorArgs, reply *FindSuccessorReply) error {
	for i, node := range n.successors {
		if hashString(string(node)).Cmp(args.Id) == 0 && i < len(n.successors)-1 {
			reply.Found = true
			reply.RetAddress = n.successors[i+1]
			return nil
		}
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
