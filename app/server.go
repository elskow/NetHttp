package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"log"
	"net"
	"strings"
)

// Types and Constants Definitions

type HTTPMethod string
type StatusCode string
type ContentType string

const (
	MethodGet  HTTPMethod = "GET"
	MethodPost HTTPMethod = "POST"

	StatusOK                  StatusCode = "HTTP/1.1 200 OK"
	StatusNotFound            StatusCode = "HTTP/1.1 404 Not Found"
	StatusInternalServerError StatusCode = "HTTP/1.1 500 Internal Server Error"
	StatusCreated             StatusCode = "HTTP/1.1 201 Created"
	StatusMethodNotAllowed    StatusCode = "HTTP/1.1 405 Method Not Allowed"

	ContentTypePlainText       ContentType = "text/plain"
	ContentTypeOctetStream     ContentType = "application/octet-stream"
	ContentTypeApplicationJSON ContentType = "application/json"
)

// Route Handler

type HandlerFunc func(conn net.Conn, request *HTTPRequest, params map[string]string)

type Server struct {
	port   string
	routes map[string]HandlerFunc
}

func (s *Server) HandleFunc(path string, handlerFunc HandlerFunc) {
	s.routes[path] = handlerFunc
}

type HTTPRequest struct {
	Method  HTTPMethod
	Path    string
	Headers map[string]string
	Body    string
}

// Server Handler

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

	for route, handler := range s.routes {
		params := make(map[string]string)
		if s.matchRoute(request.Path, route, params) {
			handler(conn, request, params)
			return
		}
	}

	s.sendResponse(conn, StatusNotFound, ContentTypePlainText, "", "", false)
}

func (s *Server) matchRoute(requestPath, route string, params map[string]string) bool {
	routeParts := strings.Split(route, "/")
	pathParts := strings.Split(requestPath, "/")

	if len(routeParts) != len(pathParts) {
		return false
	}

	for i, part := range routeParts {
		if strings.HasPrefix(part, ":") {
			paramName := part[1:]
			params[paramName] = pathParts[i]
		} else if part != pathParts[i] {
			return false
		}
	}

	return true
}

// Response and Request Handler

// Parse the request from the client.
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Messages#http_requests
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

	body, err := s.parseBody(reader, headers)
	if err != nil {
		return nil, err
	}

	return &HTTPRequest{
		Method:  method,
		Path:    path,
		Headers: headers,
		Body:    body,
	}, nil
}

func (s *Server) parseRequestLine(requestLine string) (HTTPMethod, string, error) {
	parts := strings.Split(strings.TrimSpace(requestLine), " ")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("malformed request line")
	}
	method := HTTPMethod(parts[0])
	if method != MethodGet && method != MethodPost {
		return "", "", fmt.Errorf("unsupported method: %s", method)
	}
	return method, parts[1], nil
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

func (s *Server) parseBody(reader *bufio.Reader, headers map[string]string) (string, error) {
	contentLength, ok := headers["Content-Length"]
	if !ok {
		return "", nil
	}

	return s.readBody(reader, contentLength)
}

func (s *Server) readBody(reader *bufio.Reader, contentLength string) (string, error) {
	length := 0

	fmt.Sscanf(contentLength, "%d", &length)
	body := make([]byte, length)

	_, err := reader.Read(body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// Send a response to the client.
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Messages#http_responses
func (s *Server) sendResponse(conn net.Conn, status StatusCode, contentType ContentType, body, contentEncoding string, bodyIsCompressed bool) {
	var bodyBytes []byte
	headers := fmt.Sprintf("%s\r\nContent-Type: %s\r\n", status, contentType)

	if bodyIsCompressed && contentEncoding == "gzip" {
		headers += fmt.Sprintf("Content-Encoding: %s\r\n", contentEncoding)
		var b bytes.Buffer
		gz := gzip.NewWriter(&b)
		defer gz.Close()
		if _, err := gz.Write([]byte(body)); err != nil {
			log.Printf("Failed to compress body: %v", err)
			return
		}
		if err := gz.Close(); err != nil {
			log.Printf("Failed to close gzip writer: %v", err)
			return
		}
		bodyBytes = b.Bytes()
	} else {
		bodyBytes = []byte(body)
	}

	headers += fmt.Sprintf("Content-Length: %d\r\n\r\n", len(bodyBytes))
	if _, err := conn.Write([]byte(headers)); err != nil {
		log.Printf("Failed to write headers: %v", err)
		return
	}
	if _, err := conn.Write(bodyBytes); err != nil {
		log.Printf("Failed to write body: %v", err)
	}
}
