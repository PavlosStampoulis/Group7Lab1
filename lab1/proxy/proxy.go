// curl -X GET http://localhost:1122/files/cat.jpg -x http://localhost:1234/ -o cat.jpg

package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

type Message struct {
	request string
	host    string
	other   string
}

func main() {
	var port string
	if len(os.Args) > 1 {
		port = os.Args[1]
	} else if port == "" {
		fmt.Print("Enter port to listen to: ")
		fmt.Scan(&port)
	}
	fmt.Println("Server is running on port ", port)

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error handling listen if")
	}
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error handling in connection if")
		}
		go handleConnection(conn)
	}
}

func handleConnection(cli_conn net.Conn) {
	defer cli_conn.Close()

	buf := make([]byte, 1024)
	buf_l, err := cli_conn.Read(buf)

	if err != nil {
		fmt.Println("Error handling in read client")
	}
	client_r := Message{}
	client_r.extractMessage(buf, buf_l)
	client_r.checkReqFormat()
	serv_conn, e := net.Dial("tcp", client_r.host)
	if e != nil {
		fmt.Println("Error handling in dial:" + client_r.host + ".")
	}
	defer serv_conn.Close()

	println("rq:" + client_r.request + client_r.other)
	rq := []byte(client_r.request + client_r.other)
	serv_conn.Write(rq)
	io.Copy(cli_conn, serv_conn)

}

func (client_r *Message) extractMessage(buf []byte, buf_l int) {
	message := string(buf[:buf_l])
	fmt.Print("\nrecieved:" + message + "\n")
	ms_lines := strings.Split(message, "\r\n")
	fmt.Print(ms_lines[:])
	h_to_edit := []string{"Proxy-Connection:", "Proxy-Authorization:"}

	//might use things from other later:
	for inx, ln := range ms_lines {
		if inx == 0 {
			client_r.request = ln + "\r\n"
		} else if inx == 1 {
			client_r.host = strings.TrimSpace(strings.TrimLeft(strings.Clone(ln), "Host:"))
		} else {
			toAdd := true
			for _, str := range h_to_edit {
				if strings.HasPrefix(ln, str) {
					if str == h_to_edit[0] {
						client_r.other = client_r.other + strings.TrimLeft(ln, "Proxy-") + "\r\n"
					}
					toAdd = false
				}
			}
			if toAdd {
				fmt.Println(ln)
				client_r.other = client_r.other + ln + "\r\n"
			}

		}

	}
	fmt.Printf("\nRequest " + client_r.request + "\n")
	fmt.Printf("Host " + client_r.host + "\n")
	fmt.Printf("Other " + client_r.other + "\n")

}

func (client_r *Message) checkReqFormat() {
	//check formatting of Request
	request := strings.TrimSpace(client_r.request)
	requestSplits := strings.Split(request, " ")
	requestSplits[0] = strings.ToUpper(requestSplits[0])
	if requestSplits[0] != "GET" {
		fmt.Println("Error handling for non GET or format")
		return
	}
	if requestSplits[2] != "HTTP/1.1" {
		fmt.Println("Error handling for HTTP version or format")
		return
	}

	index := strings.Index(requestSplits[1], client_r.host)
	if index != -1 {
		requestSplits[1] = requestSplits[1][index+len(client_r.host):]
	}

	client_r.request = strings.Join(requestSplits, " ") + "\r\n"
}
