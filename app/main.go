package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

var baseDir string

func connHandler(conn net.Conn) {
	defer conn.Close()

	reader := make([]byte, 4096)

	for {
		n, err := conn.Read(reader)
		if err != nil {
			log.Println("Connection closed or error:", err)
			return
		}

		request := string(reader[:n])
		fmt.Println("Full Request:\n", request)

		lines := strings.Split(request, "\r\n")
		if len(lines) == 0 {
			continue
		}

		requestLine := lines[0]
		var userAgentContent, encodingScheme, connectionHeader string

		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "User-Agent:") {
				userAgentContent = strings.TrimPrefix(line, "User-Agent: ")
			}
			if strings.HasPrefix(line, "Accept-Encoding:") {
				encodingScheme = strings.TrimPrefix(line, "Accept-Encoding: ")
			}
			if strings.HasPrefix(line, "Connection:") {
				connectionHeader = strings.ToLower(strings.TrimPrefix(line, "Connection: "))
			}
		}

		parts := strings.Split(requestLine, " ")
		if len(parts) < 2 {
			continue
		}

		method := parts[0]
		url := parts[1]

		shouldClose := connectionHeader == "close"
		connectionHeaderResp := ""
		if shouldClose {
			connectionHeaderResp = "Connection: close\r\n"
		} else {
			connectionHeaderResp = "Connection: keep-alive\r\n"
		}

		switch {
		case url == "/" && method == "GET":
			conn.Write([]byte("HTTP/1.1 200 OK\r\n" + connectionHeaderResp + "\r\n"))

		case strings.HasPrefix(url, "/echo/") && method == "GET":
			content := strings.TrimPrefix(url, "/echo/")
			hasGzip := strings.Split(encodingScheme, ", ")
			for _, scheme := range hasGzip {
				if scheme == "gzip" {
					var buf bytes.Buffer
					gz := gzip.NewWriter(&buf)
					gz.Write([]byte(content))
					gz.Close()
					compressed := buf.Bytes()
					contentLength := strconv.Itoa(len(compressed))
					response := "HTTP/1.1 200 OK\r\nContent-Encoding: gzip\r\nContent-Type: text/plain\r\nContent-Length: " + contentLength + "\r\n" + connectionHeaderResp + "\r\n"
					conn.Write([]byte(response))
					conn.Write(compressed)
					if shouldClose {
						return
					}
					goto continueLoop
				}
			}
			contentLength := strconv.Itoa(len(content))
			response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + contentLength + "\r\n" + connectionHeaderResp + "\r\n" + content
			conn.Write([]byte(response))

		case url == "/user-agent" && method == "GET":
			contentLength := strconv.Itoa(len(userAgentContent))
			response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + contentLength + "\r\n" + connectionHeaderResp + "\r\n" + userAgentContent
			conn.Write([]byte(response))

		case strings.HasPrefix(url, "/files/") && method == "GET":
			fileName := strings.TrimPrefix(url, "/files/")
			filePath := fmt.Sprintf("%s/%s", baseDir, fileName)
			fileContent, err := os.ReadFile(filePath)
			if err != nil {
				conn.Write([]byte("HTTP/1.1 404 Not Found\r\n" + connectionHeaderResp + "\r\n"))
				if shouldClose {
					return
				}
				goto continueLoop
			}
			contentLength := strconv.Itoa(len(fileContent))
			response := "HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: " + contentLength + "\r\n" + connectionHeaderResp + "\r\n" + string(fileContent)
			conn.Write([]byte(response))

		case strings.HasPrefix(url, "/files/") && method == "POST":
			fileName := strings.TrimPrefix(url, "/files/")
			filePath := fmt.Sprintf("%s/%s", baseDir, fileName)
			body := strings.SplitN(request, "\r\n\r\n", 2)
			if len(body) < 2 {
				conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n" + connectionHeaderResp + "\r\n"))
				if shouldClose {
					return
				}
				goto continueLoop
			}
			content := []byte(body[1])
			err := os.WriteFile(filePath, content, 0644)
			if err != nil {
				conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n" + connectionHeaderResp + "\r\n"))
				if shouldClose {
					return
				}
				goto continueLoop
			}
			conn.Write([]byte("HTTP/1.1 201 Created\r\n" + connectionHeaderResp + "\r\n"))

		default:
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n" + connectionHeaderResp + "\r\n"))
		}

		if shouldClose {
			return
		}

	continueLoop:
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