package dlab

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Arguments struct {
	IpAdress                 string // cur. node address
	Port                     int    // cur. node port
	JoinIpAdress             string // join node adress
	JoinPort                 int64  // join node port
	StabilizeInterval        int64  // stabilize() after 'StabilizeInterval' milliseconds
	FixFingersInterval       int64  // fixFingers() after 'FixFingersInterval' milliseconds
	CheckpredecessorInterval int64  // checkPredecessor() after 'CheckpredecessorInterval' milliseconds
	NumSuccessors            int64
	Identifier               string
}

func ReadArgsConfigureChord() *Arguments {
	var args *Arguments
	var userInput string

	fmt.Println("Usage: chord -a (IP) -p (PORT) [--ja (JOINIP) --jp (JOINPORT)] --ts (STABILIZEINTERVAL) --tff (FIXFINGERSINTERVAL) --tcp (CHECKPREDECESSORINTERVAL) -r (NUMSUCCESSORS) [-i (IDENTIFIER)]")
	fmt.Println("Examples:")
	fmt.Println("Start a new ring: chord -a 128.8.126.63 -p 4170 --ts 30000 --tff 10000 --tcp 30000 -r 4")
	fmt.Println("Join existing Chord ring: chord -a 128.8.126.63 -p 4171 --ja 128.8.126.63 --jp 4170 --ts 30000 --tff 10000 --tcp 30000 -r 4")

	reader := bufio.NewReader(os.Stdin)
	userInput, _ = reader.ReadString('\n')
	userInput = strings.TrimSpace(userInput)
	splitUserInput := strings.Fields(userInput)
	args = &Arguments{}
	if !args.ValidateArgs(splitUserInput) {
		log.Fatal("Incorret input!")
		os.Exit(1)
	}
	return args

}

func (args *Arguments) ValidateArgs(userInput []string) bool {
	for i := 0; i < len(userInput); i++ {
		switch userInput[i] {
		case "chord":
		case "-a":
			i++
			if !isValidIp(userInput[i]) {
				fmt.Printf("Invalid IP address: %s", userInput[i])
				return false
			}
			args.IpAdress = userInput[i]

		case "-p":
			i++
			port, err := validatePort(userInput[i])
			if err != nil {
				fmt.Printf("Invalid IP address: %s", userInput[i])
				return false
			}
			args.Port = int(port)

		case "--ja":
			i++
			if !isValidIp(userInput[i]) {
				fmt.Printf("Invalid IP address: %s", userInput[i])
				return false
			}
			args.JoinIpAdress = userInput[i]

		case "--jp":
			i++
			port, err := validatePort(userInput[i])
			if err != nil {
				fmt.Printf("Invalid Join IP address: %s", userInput[i])
				return false
			}
			args.JoinPort = port

		case "--ts":
			i++
			tsInterval, err := validateInterval(userInput[i], 60000)
			if err != nil {
				fmt.Printf("Invalid Stabilize Interval: %s", userInput[i])
				return false
			}
			args.StabilizeInterval = tsInterval

		case "--tff":
			i++
			tffInterval, err := validateInterval(userInput[i], 60000)
			if err != nil {
				fmt.Printf("Invalid FixFingersInterval: %s", userInput[i])
				return false
			}
			args.FixFingersInterval = tffInterval

		case "--tcp":
			i++
			tcpInterval, err := validateInterval(userInput[i], 60000)
			if err != nil {
				fmt.Printf("Invalid Check predecessor Interval: %s", userInput[i])
				return false
			}
			args.CheckpredecessorInterval = tcpInterval

		case "-r":
			i++
			NumPredecessor, err := validateInterval(userInput[i], 32)
			if err != nil {
				fmt.Printf("Invalid number of predecessor: %s", userInput[i])
				return false
			}
			args.NumSuccessors = NumPredecessor

		case "-i":
			i++
			if userInput[i] == "NodeAdress" {
				args.Identifier = userInput[i]
			}

			validIdentifierRegex, err := regexp.Compile("[0-9a-fA-F]*")
			if err != nil {
				fmt.Println("Error compiling regex:", err)
				return false
			}
			validIdentifier := validIdentifierRegex.MatchString(userInput[i])
			if !validIdentifier {
				fmt.Printf("Invalid Identifier (Please match [0-9a-fA-F]): %s", userInput[i])
				return false
			}
			args.Identifier = userInput[i]
		default:
			fmt.Println("Invalid input :", userInput[i])
			return false
		}
	}
	if (args.JoinIpAdress == "" && args.JoinPort != 0) || (args.JoinIpAdress != "" && args.JoinPort == 0) {
		fmt.Println("Error: --ja and --jp must both be specified or none.")
		return false
	}

	if args.CheckpredecessorInterval != 0 && args.FixFingersInterval != 0 && args.IpAdress != "" && args.NumSuccessors != 0 && args.Port != 0 && args.StabilizeInterval != 0 {
		return true

	}
	fmt.Println("Error: field missing, please look at the example inputs.")
	return false
}

func isValidIp(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil
}

func validatePort(portString string) (int64, error) {
	port, err := strconv.ParseInt(portString, 10, 64) // Convert to base 10!
	if err != nil {                                   // cannot convert string to base-10 integer
		return 0, err
	}
	if port >= 0 && port <= 65535 {
		return port, nil
	}

	return 0, err
}

func validateInterval(intervalStr string, upperIntervalLimit int64) (int64, error) {
	interval, err := strconv.ParseInt(intervalStr, 10, 64) // Convert to base 10!
	if err != nil {                                        // cannot convert string to base-10 integer
		return 0, err
	}
	if interval >= 1 && interval <= upperIntervalLimit {
		return interval, nil
	}
	return 0, err

}

func hashString(elt string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(elt))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}

func call(address string, method string, args interface{}, reply interface{}) error {
	c, err := rpc.DialHTTP("tcp", address)
	//sockname := coordinatorSock()
	//c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(method, args, reply)
	if err == nil {
		return nil
	}

	fmt.Println(err)
	return err
}

// start a thread that listens for RPCs from other nodes
func (n *Node) server() {
	rpc.Register(n)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", string(n.address))
	//sockname := coordinatorSock()
	//os.Remove(sockname)
	//l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)

}

func (n *Node) Ping(args *PingArgs, reply *PingReply) error {
	return nil
}

/* todo rename
func (n *Node) Predeccessor(args *PredecessorCall, reply *PredecessorResponse) error {

	reply.predecessor = n.predecessor
	reply.successors_successors = n.successors

	return nil
}*/
