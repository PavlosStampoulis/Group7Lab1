package dlab

import (
	"bufio"
	"fmt"
	"log"
	"net"
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

	fmt.Println("Usage: chord -a (IP) -p (PORT) [--ja (JOINIP) --jp (JOINPORT)] --ts (STABILIZEINTERVAL) --tff (FIXFINGERSINTERVAL) --tcp (CHECKPREDECESSORINTERVAL) -r (NUMSUCCESSORS) [-i (IDENTIFIER)]")
	fmt.Println("Examples:")
	fmt.Println("Start a new ring: chord -a 192.168.56.1 -p 4170 --ts 30000 --tff 10000 --tcp 30000 -r 4")
	fmt.Println("Join existing Chord ring: chord -a  192.168.56.1 -p 4171 --ja  192.168.56.1 --jp 4170 --ts 30000 --tff 10000 --tcp 30000 -r 4")

	splitUserInput := ReadLine()
	args = &Arguments{}
	if !args.ValidateArgs(splitUserInput) {
		log.Fatal("Incorret input!")
		os.Exit(1)
	}
	return args

}

func ReadLine() []string {
	var userInput string
	fmt.Print("$ ")
	reader := bufio.NewReader(os.Stdin)
	userInput, _ = reader.ReadString('\n')
	userInput = strings.TrimSpace(userInput)
	splitUserInput := strings.Fields(userInput)
	return splitUserInput
}

func (n *Node) ParseCommand() {
	reader := bufio.NewReader(os.Stdin)
	commandLine := ReadLine()
	switch commandLine[0] {
	case "help", "Help":
		fmt.Println("\nCommands: Ping (IP) (PORT), Lookup [???], StoreFile [???], PrintState, Quit ")

	case "Quit", "quit", "exit", "Exit":
		fmt.Println("\nMoving data to successor: " + n.successors[0])
		//TODO: call(string(n.successors[0]), "MoveAll", n.Bucket, &struct{}{})
		fmt.Println("\nTerminating Node")
		os.Exit(0)

	case "GET", "get", "Get":
		fmt.Print("Enter the name of the file you want to get: ")
		fileName, _ := reader.ReadString('\n')
		fileName = strings.TrimSpace(fileName)
		err := GetFile(fileName, n)
		if err != nil {
			fmt.Println("Couldnt get file, ", err)
		}
		fmt.Println("Get file done!")

	case "Lookup", "lookup": //Finds who has a specific key
		if len(commandLine) != 2 {
			fmt.Print("Enter the key to lookup: ")
			keyToFind, _ := reader.ReadString('\n')
			fmt.Println(keyToFind)
			Lookup(keyToFind, n)
		}
	case "StoreFile", "storefile":
		fmt.Println("Enter name of file to store:")
		fileName, _ := reader.ReadString('\n')
		err := FindAndstoreFile(fileName, n)
		if err != nil {
			fmt.Print(err)
		}
		fmt.Println("Store file done!")

	case "PrintState", "printstate":
		n.printState()

	case "Ping", "ping":
		if len(commandLine) != 3 {
			fmt.Print("Enter IP: ")
			ip, _ := reader.ReadString('\n')
			fmt.Print("Enter port: ")
			port, _ := reader.ReadString('\n')
			commandLine = append(commandLine, ip, port)
		}
		args := PingArgs{}
		reply := PongReply{}
		ok := isValidIp(commandLine[1])
		if !ok {
			fmt.Printf("Invalid IP: %s\n", commandLine[1])
			return
		}
		_, err := validatePort(commandLine[2])
		if err != nil {
			fmt.Printf("Invalid Port: %s\n", commandLine[2])
			return
		}
		err = call(commandLine[1]+":"+commandLine[2], "Node.Ping", &args, &reply)
		if err != nil {
			fmt.Println("Couldnt ping...:", err)
			return
		}
		fmt.Println("Pong! :D")

	default:
		fmt.Println("\nCommands: Ping (IP) (PORT), Lookup [???], StoreFile [???], PrintState, Quit ")
	}

}

func (args *Arguments) ValidateArgs(userInput []string) bool {
	for i := 0; i < len(userInput); i++ {
		switch userInput[i] {
		case "chord":
		case "-a":
			i++
			if !isValidIp(userInput[i]) {
				fmt.Printf("Invalid IP address: %s\n", userInput[i])
				return false
			}
			args.IpAdress = userInput[i]

		case "-p":
			i++
			port, err := validatePort(userInput[i])
			if err != nil {
				fmt.Printf("Invalid Port: %s\n", userInput[i])
				return false
			}
			args.Port = int(port)

		case "--ja":
			i++
			if !isValidIp(userInput[i]) {
				fmt.Printf("Invalid Join IP address: %s\n", userInput[i])
				return false
			}
			args.JoinIpAdress = userInput[i]

		case "--jp":
			i++
			port, err := validatePort(userInput[i])
			if err != nil {
				fmt.Printf("Invalid Join Port: %s\n", userInput[i])
				return false
			}
			args.JoinPort = port

		case "--ts":
			i++
			tsInterval, err := validateInterval(userInput[i], 60000)
			if err != nil {
				fmt.Printf("Invalid Stabilize Interval: %s\n", userInput[i])
				return false
			}
			args.StabilizeInterval = tsInterval

		case "--tff":
			i++
			tffInterval, err := validateInterval(userInput[i], 60000)
			if err != nil {
				fmt.Printf("Invalid FixFingersInterval: %s\n", userInput[i])
				return false
			}
			args.FixFingersInterval = tffInterval

		case "--tcp":
			i++
			tcpInterval, err := validateInterval(userInput[i], 60000)
			if err != nil {
				fmt.Printf("Invalid Check predecessor Interval: %s\n", userInput[i])
				return false
			}
			args.CheckpredecessorInterval = tcpInterval

		case "-r":
			i++
			NumPredecessor, err := validateInterval(userInput[i], 32)
			if err != nil {
				fmt.Printf("Invalid number of successors: %s\n", userInput[i])
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
				fmt.Printf("Invalid Identifier (Please match [0-9a-fA-F]): %s\n", userInput[i])
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

func (n *Node) printState() {
	fmt.Println("\n\n Printing state for node: " + n.Name)
	fmt.Println("Adress: ", n.address)
	fmt.Println("Predecessor: ", n.predecessor)
	fmt.Println("Identifier: ", n.Id)

	fmt.Println("Node's successors:")
	for i, node := range n.successors {
		nid := hashString(string(node))
		nid.Mod(nid, hashMod)
		fmt.Printf("Successor nr %d: %s ,id:%s\n", i+1, string(node), nid.String())
	}

	fmt.Println("Node's Finger Table:")
	for i, entr := range n.fingerTable {
		fmt.Printf("Finger %d:     Adress: %s\n", i+1, entr)
	}

	fmt.Println("Node's Bucket:")
	for key, val := range n.Bucket {
		fmt.Printf("Key: %d , Value: %s\n", key, val)
	}

}
