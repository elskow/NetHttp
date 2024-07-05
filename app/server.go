package main

import (
	"log"
	"net"
	"strconv"
	"strings"
)

const (
	CLRF = "\r\n"
)

// Response codes
const (
	OK        = "HTTP/1.1 200 OK"
	NOT_FOUND = "HTTP/1.1 404 Not Found"
)

// URL path
const (
	RootPath = "/"
	EchoPath = "/echo"
)

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Server started on :4221")

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)

	if err != nil {
		log.Println(err)
		return
	}

	if n == 0 {
		return
	}

	req := string(buf[:n])
	path := extractUrlPath(req)
	log.Println("Request path:", path)

	if path == RootPath {
		conn.Write([]byte(OK + CLRF + CLRF))
	} else if strings.HasPrefix(path, EchoPath) {
		responseBody := path[len(EchoPath)+1:]

		conn.Write([]byte(OK + CLRF))
		conn.Write([]byte("Content-Type: text/plain" + CLRF))
		conn.Write([]byte("Content-Length: " + strconv.Itoa(len(responseBody)) + CLRF + CLRF))
		conn.Write([]byte(responseBody))

	} else {
		conn.Write([]byte(NOT_FOUND + CLRF + CLRF))
	}
}

func extractUrlPath(req string) string {
	return strings.Split(req, " ")[1]
}
