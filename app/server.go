package main

import (
	"log"
	"net"
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

	response := "HTTP/1.1 200 OK\r\n\r\n"
	conn.Write([]byte(response))
	log.Printf("Response: %s", response)
}
