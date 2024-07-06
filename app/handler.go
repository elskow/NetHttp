package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func (s *Server) handleIndex(conn net.Conn, _ *HTTPRequest, _ map[string]string) {
	s.sendResponse(conn, "HTTP/1.1 200 OK", "text/plain", "", "", false)
}

func (s *Server) handleUserAgent(conn net.Conn, request *HTTPRequest, _ map[string]string) {
	userAgent := request.Headers["User-Agent"]
	s.sendResponse(conn, "HTTP/1.1 200 OK", "text/plain", userAgent, "", false)
}

func (s *Server) handleEchoMessage(conn net.Conn, request *HTTPRequest, params map[string]string) {
	message := params["message"]
	acceptEncoding := request.Headers["Accept-Encoding"]
	encodings := strings.Split(acceptEncoding, ",")
	gzipSupported := false

	for _, encoding := range encodings {
		if strings.TrimSpace(encoding) == "gzip" {
			gzipSupported = true
			break
		}
	}

	if gzipSupported {
		s.sendResponse(conn, "HTTP/1.1 200 OK", "text/plain", message, "gzip", true)
	} else {
		s.sendResponse(conn, "HTTP/1.1 200 OK", "text/plain", message, "", false)
	}
}

func (s *Server) handleFiles(conn net.Conn, request *HTTPRequest, params map[string]string) {
	method := request.Method
	filename := params["filename"]
	filePath := fmt.Sprintf("%s%s", directoryFlag, filename)

	switch method {

	case "GET":
		log.Printf("Reading file: %s", filePath)

		content, err := os.ReadFile(filePath)
		if err != nil {
			s.sendResponse(conn, "HTTP/1.1 404 Not Found", "text/plain", "", "", false)
			return
		}

		s.sendResponse(conn, "HTTP/1.1 200 OK", "application/octet-stream", string(content), "", false)

	case "POST":
		log.Printf("Writing file: %s", filePath)

		body := request.Body
		log.Printf("Body: %s", body)
		err := os.WriteFile(filePath, []byte(body), 0644)
		if err != nil {
			log.Printf("Error writing file: %s", err)
			s.sendResponse(conn, "HTTP/1.1 500 Internal Server Error", "text/plain", "", "", false)
			return
		}

		writtenContent, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Error reading back the written file: %s", err)
			s.sendResponse(conn, "HTTP/1.1 500 Internal Server Error", "text/plain", "", "", false)
			return
		}

		s.sendResponse(conn, "HTTP/1.1 201 Created", "application/octet-stream", string(writtenContent), "", false)

	default:
		s.sendResponse(conn, "HTTP/1.1 405 Method Not Allowed", "text/plain", "", "", false)
	}
}
