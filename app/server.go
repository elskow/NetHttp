package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

var directoryFlag string

func init() {
	flag.StringVar(&directoryFlag, "directory", "/tmp", "directory to create files in")
	flag.Parse()
}

type HandlerFunc func(conn net.Conn, request *HTTPRequest, params map[string]string)

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

	server.HandleFunc("/", func(conn net.Conn, request *HTTPRequest, params map[string]string) {
		server.sendResponse(conn, "HTTP/1.1 200 OK", "text/plain", "", "", false)
	})
	server.HandleFunc("/echo/:message", func(conn net.Conn, request *HTTPRequest, params map[string]string) {
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
			server.sendResponse(conn, "HTTP/1.1 200 OK", "text/plain", message, "gzip", true)
		} else {
			// Respond without compression
			server.sendResponse(conn, "HTTP/1.1 200 OK", "text/plain", message, "", false)
		}
	})
	server.HandleFunc("/user-agent", func(conn net.Conn, request *HTTPRequest, params map[string]string) {
		userAgent := request.Headers["User-Agent"]
		server.sendResponse(conn, "HTTP/1.1 200 OK", "text/plain", userAgent, "", false)
	})
	server.HandleFunc("/files/:filename", func(conn net.Conn, request *HTTPRequest, params map[string]string) {
		method := request.Method

		switch method {

		case "GET":
			filename := params["filename"]
			filePath := fmt.Sprintf("%s%s", directoryFlag, filename)
			log.Printf("Reading file: %s", filePath)

			content, err := os.ReadFile(filePath)
			if err != nil {
				server.sendResponse(conn, "HTTP/1.1 404 Not Found", "text/plain", "", "", false)
				return
			}

			server.sendResponse(conn, "HTTP/1.1 200 OK", "application/octet-stream", string(content), "", false)

		case "POST":
			filename := params["filename"]
			filePath := fmt.Sprintf("%s%s", directoryFlag, filename)
			log.Printf("Writing file: %s", filePath)

			body := request.Body
			log.Printf("Body: %s", body)
			err := os.WriteFile(filePath, []byte(body), 0644)
			if err != nil {
				log.Printf("Error writing file: %s", err)
				server.sendResponse(conn, "HTTP/1.1 500 Internal Server Error", "text/plain", "", "", false)
				return
			}

			writtenContent, err := os.ReadFile(filePath)
			if err != nil {
				log.Printf("Error reading back the written file: %s", err)
				server.sendResponse(conn, "HTTP/1.1 500 Internal Server Error", "text/plain", "", "", false)
				return
			}

			server.sendResponse(conn, "HTTP/1.1 201 Created", "application/octet-stream", string(writtenContent), "", false)

		default:
			server.sendResponse(conn, "HTTP/1.1 405 Method Not Allowed", "text/plain", "", "", false)
		}
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

	for route, handler := range s.routes {
		params := make(map[string]string)
		if s.matchRoute(request.Path, route, params) {
			handler(conn, request, params)
			return
		}
	}

	s.sendResponse(conn, "HTTP/1.1 404 Not Found", "text/plain", "", "", false)
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

func (s *Server) sendResponse(conn net.Conn, statusLine, contentType, body string, contentEncoding string, bodyIsCompressed bool) {
	headers := fmt.Sprintf("%s\r\nContent-Type: %s\r\n", statusLine, contentType)
	if bodyIsCompressed {
		headers += fmt.Sprintf("Content-Encoding: %s\r\n", contentEncoding)
	}

	headers += fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	conn.Write([]byte(headers + body))
}

func (s *Server) HandleFunc(path string, handlerFunc HandlerFunc) {
	s.routes[path] = handlerFunc
}
