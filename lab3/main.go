package main

import "fmt"

func main() {
	Arguments := ReadArgsConfigureChord()
	fmt.Printf("%+v\n", Arguments)

	if Arguments.JoinIpAdress != "" && Arguments.JoinPort != 0 {
		//config.JoinExistingChord() //TODO: :D
		//join()
	} else {
		node := NewNode(Arguments)
		node.Create()

	}
}
