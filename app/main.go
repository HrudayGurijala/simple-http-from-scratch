package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

var _ = net.Listen
var _ = os.Exit

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
	if len(lines) > 0 {
		requestLine := lines[0]
		var userAgentContent string
		for i := 1; i < len(lines); i++ {
			if strings.Contains(lines[i], "User-Agent:") {
				userAgentLine := strings.Split(lines[i], ": ")
				userAgentContent = userAgentLine[1]
				break
			}
		}

		parts := strings.Split(requestLine, " ")
		if len(parts) >= 2 {
			method := parts[0]
			url := parts[1]
			fmt.Printf("Method: %s, URL: %s\n", method, url)

			if url == "/" {
				conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
				return
			} else if strings.HasPrefix(url, "/echo/") {
				content := strings.TrimPrefix(url, "/echo/")
				contentLength := strconv.Itoa(len(content))
				response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + contentLength + "\r\n\r\n" + content
				conn.Write([]byte(response))
				return
			} else if url == "/user-agent" {
				contentLength := strconv.Itoa(len(userAgentContent))
				response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + contentLength + "\r\n\r\n" + userAgentContent
				conn.Write([]byte(response))
				return
			}
		}
	}

	// Fallback response
	conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
}

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	var conn net.Conn
	conn, err = l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}
	fmt.Println(conn)

	connHandler(conn)
}
