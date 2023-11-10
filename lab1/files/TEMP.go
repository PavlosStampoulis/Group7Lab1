package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"slices"
	"strings"
)

const (
	maxGoRoutines         = 10
	maxBufferSize         = 1024
	maxAcceptedExtensions = []string{"html", "txt", "gif", "jpeg", "jpg", "css"}
)

type serverConfig struct {
	Port          string
	MaxGoRoutines int
}

type fileHandler struct {
	c            net.Conn
	requestSplit []string
	boundary     string
	extension    string
	inFilePart   bool
	fileName     string
	fileContent  []byte
}

type fileValidation struct {
	requestSplit []string
	method       string
}

func main() {
	config := readServerConfig()
	startServer(config)
}

func readServerConfig() serverConfig {
	var config serverConfig

	fmt.Print("Enter port to listen to: ")
	fmt.Scan(&config.Port)
	config.MaxGoRoutines = maxGoRoutines

	fmt.Println("Server is running on port", config.Port)
	return config
}

func startServer(config serverConfig) {
	listener, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		log.Fatal("Couldn't start the server:", err)
	}
	defer listener.Close()

	goRoutineLimit := make(chan struct{}, config.MaxGoRoutines)

	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		goRoutineLimit <- struct{}{}
		go handleConnection(connection, goRoutineLimit)
	}
}

func handleConnection(c net.Conn, goRoutineLimit chan struct{}) {
	defer func() {
		c.Close()
		<-goRoutineLimit
	}()

	buffer := make([]byte, maxBufferSize)
	bufferLength, err := c.Read(buffer)
	if err != nil {
		handleError(c, "Couldn't read data from the client: "+err.Error(), 500)
		return
	}

	request := string(buffer[:bufferLength])
	requestSplit := strings.Split(request, "\r\n")
	method := strings.Fields(requestSplit[0])[0]

	validation := fileValidation{
		requestSplit: requestSplit,
		method:       method,
	}

	if !validateRequest(validation) {
		handleError(c, "Bad Request", 400)
		return
	}

	handler := fileHandler{
		c:            c,
		requestSplit: requestSplit,
	}

	if method == "GET" {
		getHandler(handler)
	} else if method == "POST" {
		postHandler(handler)
	} else {
		handleError(c, method+" is not implemented", 501)
	}
}

func validateRequest(validation fileValidation) bool {
	if len(validation.requestSplit) < 1 {
		log.Println("Too short request.")
		return false
	}

	firstParts := strings.Fields(validation.requestSplit[0])
	if len(firstParts) != 3 {
		log.Println("First part is not 3 parts long.")
		return false
	}

	if firstParts[2] != "HTTP/1.1" {
		log.Println("Wrong HTTP version")
		return false
	}

	if validation.method == "GET" {
		return true
	} else if validation.method == "POST" {
		for _, header := range validation.requestSplit {
			if strings.HasPrefix(header, "Content-Type:") {
				contentType := strings.TrimPrefix(header, "Content-Type:")
				contentType = strings.TrimSpace(contentType)
				if !strings.Contains(contentType, "multipart/form-data;") {
					log.Println("Please post a file!")
					return false
				}
			}
		}
	}

	return true
}

func extractBoundary(requestSplit []string) string {
	for _, header := range requestSplit {
		if strings.HasPrefix(header, "Content-Type:") && strings.Contains(header, "multipart/form-data") {
			parts := strings.Split(header, ";")
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

func postHandler(handler fileHandler) {
	// POST request handling logic
}

func getHandler(handler fileHandler) {
	filePath := strings.Fields(handler.requestSplit[0])[1]
	fileSplit := strings.Split(filePath, ".")
	fileExtension := fileSplit[len(fileSplit)-1]

	if slices.Contains(maxAcceptedExtensions, fileExtension) {
		if validPathReturnFile(filePath, handler.c, fileExtension) {
			log.Println("Successfully sent the file")
		}
	} else {
		handleError(handler.c, "Bad Request", 400)
	}
}

func validPathReturnFile(filePath string, c net.Conn, fileExtension string) bool {
	if strings.HasPrefix(filePath, "/files/") {
		fileNameSplit := strings.Split(filePath, "/")
		fileName := fileNameSplit[len(fileNameSplit)-1]
		files, err := os.ReadDir("./files")
		if err != nil {
			log.Println("Couldn't access the './files' folder", err)
			handleError(c, "Internal Error", 500)
			return false
		}

		for _, fileNameTemp := range files {
			if fileNameTemp.Name() == fileName {
				requestedFile, err := os.Open("." + filePath)
				if err != nil {
					log.Println("Couldn't access the file", err)
					handleError(c, "Internal Error", 500)
				}
				contentType := getContentType(fileExtension)
				response := "HTTP/1.1 200 OK\r\n" + "Content-Type: " + contentType + "\r\n\r\n"
				io.WriteString(c, response)
				io.Copy(c, requestedFile)
				return true
			}
		}
		handleError(c, "Not Found", 404)
	} else {
		log.Println("Not allowed to access this directory")
		handleError(c, "Bad Request", 400)
	}

	return false
}

func handleError(c net.Conn, message string, status int) {
	io.WriteString(c, "HTTP/1.1 "+fmt.Sprintf("%d", status)+" "+message+"\r\nContent-Type: text/plain\r\n\r\n"+message)
	log.Println(message)
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
		log.Println("ContentType:", contentType)
		return contentType
	}
	return "application/octet-stream"
}
