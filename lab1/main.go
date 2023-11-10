/*
for GET:
$ curl -v -o password.html http://localhost:1122/files/password.html

for POST:
curl -v -X POST -F "file=@cat2.jpg" http://localhost:1122/files
*/
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	maxGoRoutines = 10
	bufferSize    = 1024
)

type file struct {
	extension   string
	inFilePart  bool
	name        string
	content     []byte
	contentType string
}

type serverSettings struct {
	port          string
	maxGoRoutines int
}

func main() {

	server := configureServerSettings()
	runServer(server)

}

func configureServerSettings() serverSettings {
	var server serverSettings
	server.maxGoRoutines = maxGoRoutines
	if len(os.Args) > 1 {
		server.port = os.Args[1]
	} else if server.port == "" {
		fmt.Print("Enter port to listen to: ")
		fmt.Scan(&server.port)
	}
	fmt.Println("Server is running on port ", server.port)

	return server
}

func runServer(server serverSettings) {

	listener, err := net.Listen("tcp", ":"+server.port)
	if err != nil {
		log.Fatal("Couldn't start the server:", err)
	}
	defer listener.Close() //close net.listen to save resources

	goRoutineLimit := make(chan struct{}, server.maxGoRoutines) //creates a channel that controls amount of concurrent subroutines (buffer of 10)

	for {
		connection, err := listener.Accept() // wait for a new connection
		if err != nil {
			log.Fatal(err)
			continue
		}

		goRoutineLimit <- struct{}{} //limit to 10 concurrent using the channel and wait for
		//Start a new goroutine for the connection so we can run multiple concurrently
		go connectionHandler(connection, goRoutineLimit)
	}
}

func connectionHandler(connection net.Conn, goRoutineLimit chan struct{}) {
	defer func() { // defer = execution of a function until the surrounding function returns.
		connection.Close()
		<-goRoutineLimit //leave slot for another request
	}()

	buffer := make([]byte, bufferSize)
	bufferLength, err := connection.Read(buffer)
	if err != nil {
		errorHandler(connection, 500, "Internal Server Error", "Couldn't read data from the client: "+err.Error())
		return
	}

	request := string(buffer[:bufferLength])
	requestSplit := strings.Split(request, "\r\n")
	method := strings.Fields(requestSplit[0])[0] // e.g: split "GET / HTTP/1.1"  on space to get "GET"

	if !validRequest(requestSplit, method) { // Make sure the request is properly-formated.
		errorHandler(connection, 400, "Bad Request", "Badly formatted request")
		return
	}

	switch method {
	case "GET":
		GetHandler(connection, requestSplit)
	case "POST":
		PostHandler(connection, requestSplit)
	default:
		errorHandler(connection, 501, "Not Implemented", method+" is not implemented")
	}
}

func validRequest(requestSplit []string, method string) bool {
	//at least one request line is required.
	if len(requestSplit) < 1 { //error handling; remove incorrect requests
		return false
	}

	firstParts := strings.Fields(requestSplit[0]) // split GET /files/cat1.gif HTTP/1.1 on spaces
	if len(firstParts) != 3 {                     //should  be 3 words long
		return false
	}

	//checks for proper header format, this is the only version we support
	if firstParts[2] != "HTTP/1.1" {
		return false
	}

	if method == "GET" { // the request is valid for a get
		return true

	} else if method == "POST" {
		//ensure content-Type is multipart/form-data since we only accept files
		for _, header := range requestSplit {
			if strings.HasPrefix(header, "Content-Type:") {
				contentType := strings.TrimPrefix(header, "Content-Type:")
				contentType = strings.TrimSpace(contentType)
				if !strings.Contains(contentType, "multipart/form-data;") {
					return false
				}
			}
		}
	}

	return true

}
func extractBoundary(requestSplit []string) string {
	// Extract boundary from the headers in the request:
	for _, header := range requestSplit {
		if strings.HasPrefix(header, "Content-Type: multipart/form-data") {
			// The header looks like: Content-Type: multipart/form-data; boundary=------------------------682bb65ce080def9
			parts := strings.Split(header, ";") // Splits the header into parts
			for _, part := range parts {
				if strings.HasPrefix(part, " boundary=") {
					boundary := strings.TrimSpace(strings.TrimPrefix(part, " boundary="))
					return boundary
				}
			}
		}
	}
	return ""
}

func PostHandler(connection net.Conn, requestSplit []string) {
	// requessplit: [POST /files HTTP/1.1 Host: localhost:1122 User-Agent: curl/7.78.0 Accept: * Content-Length: 6916 Content-Type: multipart/form-data; boundary=------------------------c434da834ad240a8  ]
	fileLocation := strings.Fields(requestSplit[0])[1] // Make sure post is trying to save in /files directory.
	if fileLocation != "/files" {
		errorHandler(connection, 403, "Forbidden", "Please save the file in /files and not in:"+fileLocation)
		return
	}

	//extract boundary from the headers in the reqeust:
	boundary := extractBoundary(requestSplit)
	if boundary == "" {
		errorHandler(connection, 400, "Bad Request", "Failed to extract boundary")
		return
	}

	reader := bufio.NewReader(connection)
	file, err := parseHandler(reader, boundary)
	if err != nil {
		errorHandler(connection, 500, "Internal Server Error", "Couldn't parse: "+err.Error())
		return
	}
	if file == nil {
		errorHandler(connection, 400, "Internal Server Error", "Empty file")
		return

	}
	err = SaveFile(file)
	if err != nil {
		errorHandler(connection, 500, "Internal Server Error", "Couldn't save file, "+err.Error())
		return

	} else {
		file.contentType = getContentType(file.extension)
		log.Println("Successfully saved file!")
		io.WriteString(connection, "HTTP/1.1 200 OK\r\nContent-Type: "+file.contentType+"\r\n\r\n")
	}

}

func parseHandler(reader *bufio.Reader, boundary string) (*file, error) {
	var file file
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		if strings.Contains(line, boundary) {
			if file.inFilePart {
				return &file, nil //Done reading file, now return it
			}

		} else if file.inFilePart {
			// This line contains file content. Append it to the fileContent slice.
			file.content = append(file.content, []byte(line)...)
		} else if strings.Contains(line, "Content-Disposition: form-data") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "filename=") {
					file.name = strings.Split(part, `"`)[1] // Extract the filename and remove surrounding double quotes
					if strings.Contains(file.name, "..") {
						return nil, errors.New("filename has two dots in it")
					}
				}
			}
			if file.name != "" {
				// Extract the file extension and check if it's allowed
				file.extension = strings.TrimPrefix(filepath.Ext(file.name), ".")
				validExtensions := map[string]bool{
					"html": true,
					"txt":  true,
					"gif":  true,
					"jpeg": true,
					"jpg":  true,
					"css":  true,
				}
				if !validExtensions[file.extension] {
					return nil, errors.New("invalid Extension")

				}
			}
		} else if strings.Contains(line, "Content-Type") {
			lineSplit := strings.Split(line, ":")
			contentType := lineSplit[len(lineSplit)-1]
			if contentType == "" {
				return nil, errors.New("missing Content-Type header")
			}
			file.inFilePart = true
		}
	}
}

func SaveFile(file *file) error {
	filePath := "./files/" + file.name
	fileContentNew := []byte(strings.TrimSpace(string(file.content))) //remove leading and trailing spaces and convert to byte array
	//Create the file and open it
	fileHandler, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer fileHandler.Close() //close the file when we are done!

	//add content to the file
	_, err = fileHandler.Write(fileContentNew)
	if err != nil {
		return err
	}

	return nil // suceeded!

}

func getContentType(extension string) string {
	extension = strings.TrimPrefix(extension, ".")
	extentionToContentType := map[string]string{
		"html": "text/html",
		"txt":  "text/plain",
		"gif":  "image/gif",
		"jpeg": "image/jpeg",
		"jpg":  "image/jpeg",
		"css":  "text/css",
	}

	contentType, found := extentionToContentType[extension]
	if found {
		return contentType
	}
	return "application/octet-stream" //default return if unregonized file extension, however we make sure we have correct file ext. earlier so this cant happen.
}

func GetHandler(connection net.Conn, requestSplit []string) {
	filePath := strings.Fields(requestSplit[0])[1] //e.g split "GET /example.gif HTTP/1.1" on space
	fileSplit := strings.Split(filePath, ".")
	fileExtension := fileSplit[len(fileSplit)-1] // we will select the part after the last dot "/.txt.go" (security reasons)
	acceptedExtensions := []string{"html", "txt", "gif", "jpeg", "jpg", "css"}

	if slices.Contains(acceptedExtensions, fileExtension) { //if we have a correct file extension
		if validPathReturnFile(filePath, connection, fileExtension) {
			log.Println("successfully sent the file")
		}

	} else { // bad file extension, return 400
		errorHandler(connection, 400, "Bad Request", "Bad file extension")
	}
}

func validPathReturnFile(filePath string, connection net.Conn, fileExtension string) bool {
	if strings.HasPrefix(filePath, "/files/") { //check if get request is looking for a file in the files directory
		fileNameSplit := strings.Split(filePath, "/")
		fileName := fileNameSplit[len(fileNameSplit)-1] // fileName is now e.g [- index.html - style.css]
		//read & open directory
		files, err := os.ReadDir("./files")
		if err != nil {
			errorHandler(connection, 500, "Internal Server Error", "Couldn't access the './files' folder"+err.Error())
			return false
		}

		//read file names and see if the get file is in the directory
		for _, fileNameTemp := range files {
			if fileNameTemp.Name() == fileName { //if we found the file the get requested
				requestedFile, err := os.Open("." + filePath) //read the file
				if err != nil {
					errorHandler(connection, 500, "Internal Server Error", "Couldn't access the file"+err.Error())
				}
				contentType := getContentType(fileExtension)
				response := "HTTP/1.1 200 OK\r\n" + "Content-Type: " + contentType + "\r\n\r\n"
				io.WriteString(connection, response)
				io.Copy(connection, requestedFile) //return the file
				return true
			}
		}
		errorHandler(connection, 404, "Not Found", "File doesnt exist")
	} else {
		errorHandler(connection, 400, "Bad Request", "This directory is not accessable!")
	}

	return false
}

func errorHandler(connection net.Conn, httpStatus int, httpMsg string, serverMsg string) {
	log.Println(serverMsg)
	io.WriteString(connection, "HTTP/1.1 "+fmt.Sprintf("%d", httpStatus)+" "+httpMsg+"\r\nContent-Type: text/plain\r\n\r\n"+httpMsg)
	//							version				400					   Bad Request									  Bad Request
}
