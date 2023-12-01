package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type GetNReduceArgs struct {
}

type GetNReduceReply struct {
	MapNr   int
	NReduce int
}

type AskForTaskArgs struct {
	WorkerId int
}
type AskForTaskReply struct {
	TaskId      int
	TaskFile    string
	TaskType    TaskType
	ReqFilesLoc [][]string
}

type WorkerReportsTaskDoneArgs struct {
	WorkerId int
	TaskType TaskType
	TaskId   int
}
type WorkerReportsTaskDoneReply struct {
}

type WorkerLooksForFileArgs struct {
	ReqFiles []string
}
type WorkerLooksForFileReply struct {
	ReqFilesLoc [][]string
}

// Add your RPC definitions here.

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
