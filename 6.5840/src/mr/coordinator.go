package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"

	"sync"
	"time"
)

type TaskStatus int
type TaskType int

const (
	Unbegun TaskStatus = iota
	Executing
	Done
)

const (
	ReduceTask TaskType = iota
	MapTask
	ExitTask
	SleepTask
)

type Coordinator struct {
	mu          sync.Mutex // avoid conflicts in concurrent programs.
	nMap        int        // how many map tasks are left to be done
	nReduce     int
	mapTasks    []Task
	reduceTasks []Task
}

type Task struct {
	timeStamp int
	index     int
	file      string
	Type      TaskType
	workerId  int
	status    TaskStatus
}

// Your code here -- RPC handlers for the worker to call.

func (c *Coordinator) GetNReduce(args *GetNReduceArgs, reply *GetNReduceReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	reply.NReduce = len(c.reduceTasks)

	return nil

}

func (c *Coordinator) AskForTask(args *AskForTaskArgs, reply *AskForTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	idx := -1

	if c.nMap > 0 {
		idx = c.GiveTask(MapTask, args.WorkerId)
		if idx >= 0 {
			reply.TaskFile = c.mapTasks[idx].file
			reply.TaskType = c.mapTasks[idx].Type
			reply.TaskId = c.mapTasks[idx].index
			return nil
		}
	} else if c.nReduce > 0 {
		idx = c.GiveTask(ReduceTask, args.WorkerId)
		if idx >= 0 {
			reply.TaskFile = c.reduceTasks[idx].file
			reply.TaskType = c.reduceTasks[idx].Type
			reply.TaskId = c.reduceTasks[idx].index
			return nil
		}
	} else {
		reply.TaskFile = ""
		reply.TaskType = ExitTask
		reply.TaskId = -1
		return nil
	}
	reply.TaskFile = ""
	reply.TaskType = SleepTask
	reply.TaskId = -1

	return nil
}

func (c *Coordinator) GiveTask(taskType TaskType, workerId int) int {
	var tasks *[]Task
	if taskType == MapTask {
		tasks = &c.mapTasks
	} else {
		tasks = &c.reduceTasks

	}

	for index, _ := range *tasks {
		if (*tasks)[index].status == Unbegun {
			(*tasks)[index].status = Executing
			(*tasks)[index].workerId = workerId
			(*tasks)[index].timeStamp = 10

			return index
		}
	}
	//If we didnt find any tasks to give, return a "empty" task
	return -1

}

func (c *Coordinator) WorkerReportsTaskDone(args *WorkerReportsTaskDoneArgs, reply *WorkerReportsTaskDoneReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if (args.TaskType) == MapTask {
		task := &c.mapTasks[args.TaskId]
		if task.status != Done {
			task.status = Done
			c.nMap -= 1
		}

	} else {
		task := &c.reduceTasks[args.TaskId]
		if task.status != Done {
			task.status = Done
			c.nReduce -= 1
		}

	}

	return nil
}

// tick down time left for executing tasks
func (c *Coordinator) timer() {

	for !c.Done() {
		time.Sleep(1 * time.Second)
		c.mu.Lock()
		if c.nMap != 0 { // All not maptasks done
			for idx, _ := range c.mapTasks {
				if c.mapTasks[idx].status == Executing {
					c.mapTasks[idx].timeStamp -= 1
					if c.mapTasks[idx].timeStamp == 0 {
						if c.mapTasks[idx].status != Done {
							c.mapTasks[idx].status = Unbegun
							c.mapTasks[idx].workerId = -1
						}
					}
				}
			}
		} else {
			for idx, _ := range c.reduceTasks {
				if c.reduceTasks[idx].status == Executing {
					c.reduceTasks[idx].timeStamp -= 1
					if c.reduceTasks[idx].timeStamp == 0 {
						if c.reduceTasks[idx].status != Done {
							c.reduceTasks[idx].status = Unbegun
							c.reduceTasks[idx].workerId = -1
						}

					}
				}
			}
		}
		c.mu.Unlock()
	}

}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":20796")
	//sockname := coordinatorSock()
	//os.Remove(sockname)
	//l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)

}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return (c.nMap == 0 && c.nReduce == 0)
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}
	var nMap = len(files)

	c.nMap = nMap
	c.nReduce = nReduce
	c.reduceTasks = make([]Task, 0, nReduce)
	c.mapTasks = make([]Task, 0, nMap)
	go c.timer()
	//append maptasks
	for i := 0; i < nMap; i++ {
		mapTaskNew := Task{Type: MapTask, status: Unbegun, index: i, file: files[i], workerId: -1}
		c.mapTasks = append(c.mapTasks, mapTaskNew)
	}

	//append reducetasks
	for i := 0; i < nReduce; i++ {
		reduceTaskNew := Task{Type: ReduceTask, status: Unbegun, index: i, file: strconv.Itoa(i), workerId: -1}
		c.reduceTasks = append(c.reduceTasks, reduceTaskNew)
	}
	c.server()

	return &c
}
