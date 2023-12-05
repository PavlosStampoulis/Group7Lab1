package main

import (
	"fmt"
)

func main() {
	Arguments := ReadArgsConfigureChord()
	fmt.Printf("%+v\n", Arguments)

	node := NewNode(Arguments)
	fmt.Printf("New node created: %+v\n", node)

	node.server()
	//Todo: Handle connections

	if Arguments.JoinIpAdress != "" && Arguments.JoinPort != 0 {
		//config.JoinExistingChord() //TODO: :D
		//join()
	} else { // Create a new chord ring
		node.CreateChord()
	}
}
