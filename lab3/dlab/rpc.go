package dlab

import (
	"crypto/rsa"
	"math/big"
)

type PingArgs struct {
}
type PongReply struct {
}

/*
type NumberSuccessorsCall struct {
}

	type NumberSuccessorsResponse struct {
		numberofsuccessors int64
	}
*/
type StabilizeCall struct {
}

// Response from stabilize data
type StabilizeResponse struct {
	Predecessor           NodeAddress
	Successors_successors []NodeAddress
}

type NotifyArgs struct {
	Address NodeAddress
}

type NotifyReply struct {
	Ok bool
}

type FindSuccessorArgs struct {
	Id *big.Int
}

type FindSuccessorReply struct {
	Found      bool
	RetAddress NodeAddress
}

type File struct {
	Name    string
	Id      *big.Int
	Content []byte
}

type StoreFileReply struct {
	Succeeded bool
	Error     error
}

type StoreFileArgs struct {
	File File
}

type RequestPubKeyArgs struct {
}

type RequestPubKeyReply struct {
	Key *rsa.PublicKey
}
