package mr

import (
	//"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
	"sort"
	"strings"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

var nReduce int

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {
	//println("I started")
	//send RPC to coordinator asking for a task
	// get reduce count from master
	nReduceAns, ok := getNReduce()
	okHandler(ok, "Couldn't get nReduce from coordinator")
	nReduce = nReduceAns
	for {
		taskReply, ok := askForTask()
		okHandler(ok, "Couldn't get task from coordinator")
		switch taskReply.TaskType {
		case MapTask:
			mapHandler(taskReply.TaskFile, mapf, taskReply.TaskId)
		case ReduceTask:
			reduceHandler(taskReply.TaskFile, reducef, taskReply.TaskId)
		case ExitTask: // shuts down worker
			log.Println("Worker got ExitTask, quiting worker.")
			os.Exit(0)
		case SleepTask:
			time.Sleep(1 * time.Microsecond)
		default:
		}

	}

}

func mapHandler(filePath string, mapf func(string, string) []KeyValue, taskId int) {
	// Read fileContent
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening file", err)
		return
	}
	fileContent, err := io.ReadAll(file)
	if err != nil {
		log.Println("Error reading file", err)
		return
	}
	err = file.Close()
	if err != nil {
		log.Println("Error closing file", err)
		return
	}

	// Create intermediate files
	kv := mapf(filePath, string(fileContent))

	// Specify the desired directory
	outputDirectory := "/home/ec2-user/fs1"

	// Making a temporary folder to store intermediate results
	toFile := make([][]string, nReduce)
	for _, keyv := range kv {
		Y := ihash(keyv.Key) % nReduce
		toFile[Y] = append(toFile[Y], string(keyv.Key)+":"+string(keyv.Value)+"\n")
	}

	for i, content := range toFile {
		mrfilename := fmt.Sprintf("%s/mri-%d-%d.txt", outputDirectory, taskId, i)
		tmpfile, err := os.CreateTemp(outputDirectory, "")
		if err != nil {
			log.Println(err)
			return
		}
		for _, text := range content {
			tmpfile.WriteString(text)
		}
		tmpfile.Close()

		// Move the temporary file to the final location
		err = os.Rename(tmpfile.Name(), mrfilename)
		if err != nil {
			log.Println("Error renaming", err)
			return
		}
	}

	WorkerReportsTaskDone(taskId, MapTask)
}


func reduceHandler(filePath string, reducef func(string, []string) string, taskId int) {
	//mr-X-Y.txt

	buckets, err := os.ReadDir("/home/ec2-user/fs1")
	//errorHandler(err,"Error reading folder")
	if err != nil { // Handle the error if the folder read fails
		log.Println("Error reading folder:", err)
		return
	}
	intermediate := []KeyValue{}
	for _, bucket_f := range buckets {
		bucket := bucket_f.Name()
		if !strings.HasPrefix(bucket, "mri-") {
			continue
		}
		idx := strings.LastIndex(bucket, "-")
		if idx < 0 {
			continue
		}
		bucket_nr, f := strings.CutSuffix(bucket[idx+1:], ".txt")
		if !f {
			continue
		}
		if bucket_nr == filePath {
			file, err := os.Open(bucket_f.Name())
			if err != nil { // Handle the error if the open fails
				log.Println("Error opening file:", err)
				return
			}
			buf, err := io.ReadAll(file)
			if err != nil {
				log.Println("Error reading file:", err)
				return
			}
			err = file.Close()
			if err != nil {
				log.Println("Error closing file:", err)
			}

			b_string := string(buf)
			file_lines := strings.Split(strings.TrimSpace(b_string), "\n")
			for _, ln := range file_lines {
				var tempKeyvalue KeyValue
				idx = strings.LastIndex(ln, ":")
				if idx < 0 {
					continue
				}
				value := ln[idx+1:]
				key := ln[:idx]
				tempKeyvalue.Key = key
				tempKeyvalue.Value = value
				intermediate = append(intermediate, tempKeyvalue)
			}

		}

	}

	//Added this from mrseqauential.go
	sort.Sort(ByKey(intermediate))
	oname := "mr-out-" + filePath + ".txt"

	ofile, err := os.CreateTemp("/home/ec2-user/fs1", "")
	if err != nil {
		log.Println("Error creating file:", err)
		return
	}

	//
	// call Reduce on each distinct key in intermediate[],
	// and print the result to mr-out-0.
	//
	i := 0
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducef(intermediate[i].Key, values)

		// this is the correct format for each line of Reduce output.
		_, err = fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)
		if err != nil {
			log.Println("Error writing output", err)
		}
		i = j
	}
	ofile.Close()

	err = os.Rename(ofile.Name(), "/home/ec2-user/fs1"+oname)
	WorkerReportsTaskDone(taskId, ReduceTask)
}

func getNReduce() (int, bool) {
	args := GetNReduceArgs{}
	reply := GetNReduceReply{}
	ok := call("Coordinator.GetNReduce", &args, &reply)

	return reply.NReduce, ok
}

func askForTask() (*AskForTaskReply, bool) {
	test := os.Getpid()
	args := AskForTaskArgs{test}
	reply := AskForTaskReply{}
	ok := call("Coordinator.AskForTask", &args, &reply)
	return &reply, ok

}

func WorkerReportsTaskDone(taskId int, taskType TaskType) bool {
	args := WorkerReportsTaskDoneArgs{os.Getpid(), taskType, taskId}
	reply := WorkerReportsTaskDoneReply{}
	ok := call("Coordinator.WorkerReportsTaskDone", &args, &reply)

	return ok
}

func errorHandler(err error, desc string) {
	if err != nil {
		log.Println(desc, " ", err)
	}
}
func okHandler(didWork bool, errorDesc string) {
	if !didWork {
		log.Println(errorDesc)
		return
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	c, err := rpc.DialHTTP("tcp", "54.237.95.61"+":1234")
	//sockname := coordinatorSock()
	//c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
