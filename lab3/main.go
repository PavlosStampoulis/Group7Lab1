package main

import (
	"fmt"
	"lab3/dlab"
	"net/rpc"
	"time"
)

// chord -a 192.168.56.1 -p 4170 --ts 30000 --tff 10000 --tcp 30000 -r 4
// chord -a 192.168.56.1 -p 4172 --ja 192.168.56.1 --jp 4170 --ts 30000 --tff 10000 --tcp 30000 -r 4
// Ping 192.168.56.1 4170

func main() {

	Arguments := dlab.ReadArgsConfigureChord()

	node := dlab.NewNode(Arguments)

	rpc.Register(node) // now we can call using the node

	node.Server()

	if Arguments.JoinIpAdress != "" && Arguments.JoinPort != 0 { //Join existing chord ring
		adressToJoin := fmt.Sprintf("%s:%d", Arguments.JoinIpAdress, Arguments.JoinPort)
		fmt.Println("Joining existing chord ring with address: ", adressToJoin)
		node.Join(Arguments)
	} else { // Create a new chord ring
		fmt.Println("Creating a chord with adress: ", Arguments.IpAdress, ":", Arguments.Port)
	}

	go node.TimedCalls(Arguments.SInterval, Arguments.FInterval, Arguments.CInterval)

	fmt.Println("\nCommands: Ping (IP) (PORT), Lookup [???], StoreFile [???], PrintState, Quit")
	for {
		node.ParseCommand()
		time.Sleep(time.Millisecond)
	}

}
