package main

type PingArgs struct {
}
type PingReply struct {
}

type PredecessorCall struct {
}

// Response from the predecessor check
type PredecessorResponse struct {
	predecessor           NodeAddress
	successors_successors []NodeAddress
}
