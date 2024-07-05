package main

import (
	"log"
	"net"
	"strings"
)

const (
	OK        = "HTTP/1.1 200 OK\r\n\r\n"
	NOT_FOUND = "HTTP/1.1 404 Not Found\r\n\r\n"
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

	if path == "/" {
		conn.Write([]byte(OK))
	} else {
		conn.Write([]byte(NOT_FOUND))
	}
}

func extractUrlPath(req string) string {
	return strings.Split(req, " ")[1]
}
