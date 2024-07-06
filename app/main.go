package main

import (
	"flag"
)

var directoryFlag string

func init() {
	flag.StringVar(&directoryFlag, "directory", "/tmp", "directory to create files in")
	flag.Parse()
}

func main() {
	server := NewServer("4221")
	server.setupRoutes()
	server.ListenAndServe()
}

func (s *Server) setupRoutes() {
	s.HandleFunc("/", s.handleIndex)
	s.HandleFunc("/echo/:message", s.handleEchoMessage)
	s.HandleFunc("/user-agent", s.handleUserAgent)
	s.HandleFunc("/files/:filename", s.handleFiles)
}
