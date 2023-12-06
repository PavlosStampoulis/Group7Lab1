package main

import (
	"fmt"
	"lab3/dlab"
	"time"
)

func main() {

	Arguments := dlab.ReadArgsConfigureChord()
	fmt.Printf("%+v\n", Arguments)

	node := dlab.NewNode(Arguments)
	fmt.Printf("New node created: %+v\n", node)
	//Todo: Handle connections

	//temporary
	exit := false
	for !exit {
		time.Sleep(time.Millisecond)
		node.ParseCommand()
	}

	// Make sure hand-off gets done to not lose content before
	// the node decides to exit

}
