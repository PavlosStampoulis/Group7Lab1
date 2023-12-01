package mr

import (
	//"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"net/rpc"
	"os"
	"sort"
	"strconv"
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
var mapNr int
var tmpDir string
var handledMap map[string]bool
var myId string

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
	nReduceAns, mapNrAns, ok := getNReduce()
	okHandler(ok, "Couldn't get nReduce from coordinator")
	nReduce = nReduceAns
	mapNr = mapNrAns //to know how many files to expect in reduce
	dir, err := os.MkdirTemp(".", "")
	tmpDir = dir
	if err != nil {
		log.Println("error creating folder ", err)
	}
	go fileSender()

	for {
		taskReply, ok := askForTask()
		okHandler(ok, "Couldn't get task from coordinator")
		switch taskReply.TaskType {
		case MapTask:
			mapHandler(taskReply.TaskFile, mapf, taskReply.TaskId)
		case ReduceTask:
			reduceGetFiles(taskReply.TaskFile, taskReply.ReqFilesLoc)
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

func fileSender() {
	http.HandleFunc("/", fileSendHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func fileSendHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		splitR := strings.Split(req.URL.Path, "/")
		filename := splitR[len(splitR)-1]

		files, err := os.ReadDir(tmpDir)
		if err != nil {
			log.Println("Error reading directory")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, file := range files {
			if file.Name() == filename {
				http.ServeFile(w, req, tmpDir+"/"+filename)
				return
			}

		}
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusForbidden)
}

func reduceGetFiles(fileEnding string, fileLocations [][]string) {
	for i := 0; i < mapNr; i++ {
		_, handledMap := handledMap[strconv.Itoa(i)]
		if !handledMap {
			file := "mri-" + strconv.Itoa(i) + "-" + fileEnding + ".txt"
			for _, id := range fileLocations[i] {
				ok := askForFile(file, id)
				if ok {
					break
				}
			}
		}
	}
	//add check to see all files are there
}

func askForFile(file string, id string) bool {

	tmpFile, err := os.CreateTemp(tmpDir+"/", "")
	if err != nil {
		log.Println("Error creating file", err)
		return false
	}
	resp, err := http.Get(id + "/" + file)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		log.Println(err)
		return false
	}
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		log.Println("Error copying file", err)
		return false
	}
	tmpFile.Close()
	err = os.Rename(tmpFile.Name(), tmpDir+"/"+file)
	if err != nil {
		log.Println("Error renaming", err)
		return false
	}

	return true
}

func mapHandler(filePath string, mapf func(string, string) []KeyValue, taskId int) {
	//Read fileContent
	file, err := os.Open(filePath)
	if err != nil { // Handle the error if the folder creation fails
		log.Println("Error opening file", err)
		return
	}
	fileContent, err := io.ReadAll(file)
	if err != nil { // Handle the error if the folder creation fails
		log.Println("Error reading file", err)
		return
	}
	err = file.Close()
	if err != nil { // Handle the error if file close fails
		log.Println("Error closing file", err)
		return
	}
	//Create intermediate files
	kv := mapf(filePath, string(fileContent))
	// Write ihash values to the local file
	// making a temporary folder to store in while still
	// generating the intermediate result
	if err != nil { // Handle the error if the folder creation fails
		log.Println("Error creating folder:", err)
		return
	}
	toFile := make([][]string, nReduce)
	for _, keyv := range kv {
		Y := ihash(keyv.Key) % nReduce
		toFile[Y] = append(toFile[Y], string(keyv.Key)+":"+string(keyv.Value)+"\n")
	}
	for i, content := range toFile {
		mrfilename := "mri-" + strconv.Itoa(taskId) + "-" + strconv.Itoa(i) + ".txt"
		tmpfile, err := os.CreateTemp(tmpDir+"/", "")
		if err != nil {
			log.Println(err)
		}

		for _, text := range content {
			tmpfile.WriteString(text)
		}
		tmpfile.Close()
		fmt.Println(tmpfile.Name())
		err = os.Rename(tmpfile.Name(), tmpDir+"/"+mrfilename)
		if err != nil {
			log.Println("Error renaming", err)
			return
		}
	}
	handledMap[strconv.Itoa(taskId)] = true
	WorkerReportsTaskDone(taskId, MapTask)
}

func reduceHandler(filePath string, reducef func(string, []string) string, taskId int) {
	//mr-X-Y.txt

	buckets, err := os.ReadDir(tmpDir)
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
			file, err := os.Open(tmpDir + "/" + bucket_f.Name())
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

	ofile, err := os.CreateTemp(tmpDir, "")
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

	err = os.Rename(ofile.Name(), tmpDir+"/"+oname)
	if err != nil {
		log.Println("Error renaming", err)
		return
	}
	WorkerReportsTaskDone(taskId, ReduceTask)
}

func getNReduce() (int, int, bool) {
	args := GetNReduceArgs{}
	reply := GetNReduceReply{}
	ok := call("Coordinator.GetNReduce", &args, &reply)

	return reply.NReduce, reply.MapNr, ok
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

func WorkerLooksForFile(files []string) bool {
	args := WorkerLooksForFileArgs{files}
	reply := WorkerLooksForFileReply{}
	ok := call("Coordinator.WorkerLooksForFile", &args, &reply)
	return ok
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
	c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
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
