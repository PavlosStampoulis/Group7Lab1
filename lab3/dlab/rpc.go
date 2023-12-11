package dlab

import (
	"crypto/rsa"
	"math/big"
)

type PingArgs struct {
}
type PongReply struct {
}

type StabilizeCall struct {
}

// Response from stabilize data
type StabilizeResponse struct {
	Predecessor           NodeInfo
	Successors_successors []NodeInfo
}

type NotifyArgs struct {
	Info NodeInfo
}

type NotifyReply struct {
	Ok bool
}

type FindSuccessorArgs struct {
	Id *big.Int
}

type FindSuccessorReply struct {
	Found      bool
	RetAddress NodeInfo
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
