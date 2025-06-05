package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"compress/gzip"
)

var _ = net.Listen
var _ = os.Exit

var baseDir string

func connHandler(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Println("Error reading:", err)
		return
	}

	request := string(buffer[:n])
	fmt.Println("Full Request:\n", request)

	lines := strings.Split(request, "\r\n")
	if len(lines) == 0 {
		return
	}

	requestLine := lines[0]
	var userAgentContent string
	var encodingScheme string
	for i := 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "User-Agent:") {
			userAgentLine := strings.SplitN(lines[i], ": ", 2)
			if len(userAgentLine) == 2 {
				userAgentContent = userAgentLine[1]
			}
			break
		}
	}
	for i := 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "Accept-Encoding:") {
			encodingSchemeLine := strings.SplitN(lines[i], ": ", 2)
			if len(encodingSchemeLine) == 2 {
				encodingScheme = encodingSchemeLine[1]
			}
			break
		}
	}

	parts := strings.Split(requestLine, " ")
	if len(parts) < 2 {
		return
	}

	method := parts[0]
	url := parts[1]

	fmt.Printf("Method: %s, URL: %s\n", method, url)

	switch {
	case url == "/" && method == "GET":
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

	case strings.HasPrefix(url, "/echo/") && method == "GET":
		content := strings.TrimPrefix(url, "/echo/")
		contentLength := strconv.Itoa(len(content))
		var response string
		hasGzip := strings.Split(encodingScheme,", ")

		for _,scheme := range hasGzip {
			
			if scheme == "gzip" {
				var buf bytes.Buffer
				gz := gzip.NewWriter(&buf)
				gz.Write([]byte(content))
				gz.Close()
				compressed := buf.Bytes()
	
				contentLength := strconv.Itoa(len(compressed))
				response := "HTTP/1.1 200 OK\r\nContent-Encoding: gzip\r\nContent-Type: text/plain\r\nContent-Length: " + contentLength + "\r\n\r\n"
				conn.Write([]byte(response))
				conn.Write(compressed)
				return
			}
		}

		response = "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + contentLength + "\r\n\r\n" + content
		conn.Write([]byte(response))

	case url == "/user-agent" && method == "GET":
		contentLength := strconv.Itoa(len(userAgentContent))
		response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + contentLength + "\r\n\r\n" + userAgentContent
		conn.Write([]byte(response))

	case strings.HasPrefix(url, "/files/") && method == "GET":
		fileName := strings.TrimPrefix(url, "/files/")
		filePath := fmt.Sprintf("%s/%s", baseDir, fileName)
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			log.Println("error reading file:", err)
			return
		}
		contentLength := strconv.Itoa(len(fileContent))
		response := "HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: " + contentLength + "\r\n\r\n" + string(fileContent)
		conn.Write([]byte(response))

	case strings.HasPrefix(url, "/files/") && method == "POST":
		fileName := strings.TrimPrefix(url, "/files/")
		filePath := fmt.Sprintf("%s/%s", baseDir, fileName)

		body := strings.SplitN(request, "\r\n\r\n", 2)
		if len(body) < 2 {
			conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
			return
		}
		content := []byte(body[1])

		err := os.WriteFile(filePath, content, 0644)
		if err != nil {
			conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
			log.Println("error writing file:", err)
			return
		}

		conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))

	default:
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}

func main() {

	for i, arg := range os.Args {
		if arg == "--directory" && i+1 < len(os.Args) {
			baseDir = os.Args[i+1]
			break
		}
	}

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			os.Exit(1)
		}

		go connHandler(conn)
	}
}