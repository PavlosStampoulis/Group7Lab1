package dlab

import (
	"fmt"
	"log"
	"net/rpc"
)

func call(address string, method string, args interface{}, reply interface{}) error {
	c, err := rpc.DialHTTP("tcp", address)
	//sockname := coordinatorSock()
	//c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(method, args, reply)
	if err == nil {
		return nil
	}

	fmt.Println(err)
	return err
}

func (n *Node) Ping(args *PingArgs, reply *PingReply) error {

	return nil
}

/*
func (n *Node) getNumberSuccessors(args *NumberSuccessorsCall, reply *NumberSuccessorsResponse) error {

	reply.numberofsuccessors = globalNumberSuccessors

	return nil
}*/

// NotifyReceiver: recieve notification from node believing to be our Predecessor
func (n *Node) NotifyReceiver(args *NotifyArgs, reply *NotifyReply) error {
	if (n.predecessor == "") || false { //TODO swap false for second condition
		n.predecessor = args.Address
		reply.Ok = true
		return nil
	}

	reply.Ok = false
	return nil
}

func (n *Node) StabilizeData(args *StabilizeCall, reply *StabilizeResponse) error {

	reply.numberofsuccessors = globalNumberSuccessors
	reply.address = n.address
	reply.predecessor = n.predecessor
	reply.successors_successors = n.successors

	return nil
}
