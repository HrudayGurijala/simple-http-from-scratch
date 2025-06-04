package main

import (
	"fmt"
	"log"
	"net"
	"os"
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
		parts := strings.Split(requestLine, " ")
		if len(parts) >= 2 {
			method := parts[0]
			url := parts[1]
			fmt.Printf("Method: %s, URL: %s\n", method, url)

			if url == "/" {
				conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
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
