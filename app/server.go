package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

type HandlerFunc func(conn net.Conn, request *HTTPRequest)

type Server struct {
	port   string
	routes map[string]HandlerFunc
}

type HTTPRequest struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    string
}

func main() {
	server := NewServer("4221")
	server.HandleFunc("/", func(conn net.Conn, request *HTTPRequest) {
		server.sendResponse(conn, "HTTP/1.1 200 OK", "text/plain", "")
	})
	server.HandleFunc("/echo/*", func(conn net.Conn, request *HTTPRequest) {
		pathParts := strings.Split(request.Path, "/")
		if len(pathParts) < 3 {
			server.sendResponse(conn, "HTTP/1.1 400 Bad Request", "text/plain", "Invalid path")
			return
		}
		server.sendResponse(conn, "HTTP/1.1 200 OK", "text/plain", pathParts[2])
	})
	server.HandleFunc("/user-agent", func(conn net.Conn, request *HTTPRequest) {
		userAgent := request.Headers["User-Agent"]
		server.sendResponse(conn, "HTTP/1.1 200 OK", "text/plain", userAgent)
	})

	server.ListenAndServe()
}

func NewServer(port string) *Server {
	return &Server{
		port:   port,
		routes: make(map[string]HandlerFunc),
	}
}

func (s *Server) ListenAndServe() {
	listener, err := net.Listen("tcp", "[::]:"+s.port)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()
	log.Printf("Server started on :%s", s.port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	request, err := s.parseRequest(conn)
	if err != nil {
		log.Printf("Failed to parse request: %v", err)
		return
	}

	if handler, ok := s.routes[request.Path]; ok {
		handler(conn, request)
		return
	}

	for route, handler := range s.routes {
		if strings.Contains(route, "*") {
			baseRoute := strings.Split(route, "*")[0]
			if strings.HasPrefix(request.Path, baseRoute) {
				handler(conn, request)
				return
			}
		}
	}

	s.sendResponse(conn, "HTTP/1.1 404 Not Found", "text/plain", "")
}

func (s *Server) parseRequest(conn net.Conn) (*HTTPRequest, error) {
	reader := bufio.NewReader(conn)
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	method, path, err := s.parseRequestLine(requestLine)
	if err != nil {
		return nil, err
	}

	headers, err := s.parseHeaders(reader)
	if err != nil {
		return nil, err
	}

	return &HTTPRequest{
		Method:  method,
		Path:    path,
		Headers: headers,
	}, nil
}

func (s *Server) parseRequestLine(requestLine string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(requestLine), " ")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("malformed request line")
	}
	return parts[0], parts[1], nil
}

func (s *Server) parseHeaders(reader *bufio.Reader) (map[string]string, error) {
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) < 2 {
			continue
		}
		headers[parts[0]] = parts[1]
	}
	return headers, nil
}

func (s *Server) sendResponse(conn net.Conn, statusLine, contentType, body string) {
	headers := fmt.Sprintf("%s\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n", statusLine, contentType, len(body))
	conn.Write([]byte(headers + body))
}

func (s *Server) HandleFunc(path string, handlerFunc HandlerFunc) {
	s.routes[path] = handlerFunc
}
