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

// Response from stabiliza data
type StabilizeResponse struct {
	numberofsuccessors    int64
	address               NodeAddress
	predecessor           NodeAddress
	successors_successors []NodeAddress
}

type NotifyArgs struct {
	Address NodeAddress
}

type NotifyReply struct {
	Ok bool
}
