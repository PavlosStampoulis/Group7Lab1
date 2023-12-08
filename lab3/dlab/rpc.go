package dlab

type PingArgs struct {
}
type PingReply struct {
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
	Numberofsuccessors    int64
	Address               NodeAddress
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
	Id NodeAddress
}

type FindSuccessorReply struct {
	Found      bool
	RetAddress NodeAddress
}
