package main

import (
	"fmt"
	"lab3/dlab"
	"net/rpc"
	"time"
)

func main() {

	Arguments := dlab.ReadArgsConfigureChord()

	node := dlab.NewNode(Arguments)

	rpc.Register(node) // now we can call using the node
	node.Server()

	if Arguments.JoinIpAdress != "" && Arguments.JoinPort != 0 { //Join existing chord ring
		adressToJoin := fmt.Sprintf("%s:%d", Arguments.JoinIpAdress, Arguments.JoinPort)
		fmt.Println("Joining existing chord ring with address: ", adressToJoin)
		node.Join(Arguments) //TODO: joinfile
	} else { // Create a new chord ring
		fmt.Println("Creating a chord with adress: ", Arguments.IpAdress, ":", Arguments.Port)
		node.CreateChord()
	}

	//TODO: start timers (FixFingers, Stabilize, CheckPredecesor)

	fmt.Println("\nCommands: Ping (IP) (PORT), Lookup [???], StoreFile [???], PrintState, Quit")
	for {
		node.ParseCommand()

		time.Sleep(time.Millisecond)
	}

}
