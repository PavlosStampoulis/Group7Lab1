// curl -X GET http://localhost:1122/files/cat.jpg -x http://localhost:1234/ -o cat.jpg

package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

type Message struct {
	cli_conn net.Conn
	request  string
	host     string
	other    string
}

var rq_methods = map[string]bool{
	"GET":     true,
	"POST":    false,
	"HEAD":    false,
	"PUT":     false,
	"DELETE":  false,
	"CONNECT": false,
	"OPTIONS": false,
	"TRACE":   false,
	"PATCH":   false,
}

func main() {

	//Start up server, if port not provided ask for it
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
		log.Fatal("Couldn't start the server:", err)
	}

	defer ln.Close()

	//Loop accepting new incomming connections, no preset max child processes
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal("Error accepting connection ", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(cli_conn net.Conn) {
	defer cli_conn.Close()

	//read client request
	buf := make([]byte, 1024)
	buf_l, err := cli_conn.Read(buf)

	if err != nil {
		errorHandler(cli_conn, 500, "Internal Server Error", "Couldn't read data from the client: "+err.Error())
		return
	}
	//extract and modify parts of the message to pass on to server
	client_r := Message{}
	client_r.cli_conn = cli_conn
	if !client_r.extractMessage(buf, buf_l) {
		return
	}
	if !client_r.checkReqFormat() {
		return
	}

	//connect to the requested server
	serv_conn, e := net.Dial("tcp", client_r.host)
	if e != nil {
		errorHandler(cli_conn, 502, "Bad Gateway", "Could not reach host: "+client_r.host)
		return
	}
	defer serv_conn.Close()

	//pass on request and forward reply
	rq := []byte(client_r.request + client_r.other)
	_, err = serv_conn.Write(rq)
	if err != nil {
		errorHandler(cli_conn, 504, "Gateway Timeout", "Couldn't send request to host: "+err.Error())
		return
	}

	io.Copy(cli_conn, serv_conn)

}

func (client_r *Message) extractMessage(buf []byte, buf_l int) bool {

	message := string(buf[:buf_l])

	//Splits the message into lines
	ms_lines := strings.Split(message, "\r\n")

	//So that we at least have GET and host.
	if len(ms_lines) < 2 {
		errorHandler(client_r.cli_conn, 400, "Bad Request", "Badly formatted request")
		return false
	}
	//vector to match header parts that might be included and need special interaction
	h_to_edit := []string{"Proxy-Connection:", "Proxy-Authorization:"}

	//check header line by line
	for inx, ln := range ms_lines {
		if inx == 0 {
			client_r.request = ln + "\r\n"
		} else if inx == 1 {

			if !strings.Contains(ln, "Host:") {
				errorHandler(client_r.cli_conn, 400, "Bad Request", "Badly formatted request")
				return false
			}
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
				client_r.other = client_r.other + ln + "\r\n"
			}

		}

	}

	return true
}

func (client_r *Message) checkReqFormat() bool {
	//check formatting of Request
	request := strings.TrimSpace(client_r.request)
	requestSplits := strings.Split(request, " ")
	requestSplits[0] = strings.ToUpper(requestSplits[0])
	rq_imple, exist := rq_methods[requestSplits[0]]
	if !exist {
		errorHandler(client_r.cli_conn, 400, "Bad Request", "Badly formatted request")
		return false
	} else if !rq_imple {
		errorHandler(client_r.cli_conn, 501, "Not Implemented", requestSplits[0]+" is not implemented")
		return false
	}
	if requestSplits[2] != "HTTP/1.1" {
		errorHandler(client_r.cli_conn, 505, "HTTP Version", "Not Supported")
		return false
	}

	//check if host is in request row, if so remove
	index := strings.Index(requestSplits[1], client_r.host)
	if index != -1 {
		requestSplits[1] = requestSplits[1][index+len(client_r.host):]
	}

	client_r.request = strings.Join(requestSplits, " ") + "\r\n"
	return true
}

func errorHandler(connection net.Conn, httpStatus int, httpMsg string, serverMsg string) {
	log.Println(serverMsg)
	io.WriteString(connection, "HTTP/1.1 "+fmt.Sprintf("%d", httpStatus)+" "+httpMsg+"\r\nContent-Type: text/plain\r\n\r\n"+httpMsg)
	//							version				400					   Bad Request									  Bad Request
}
