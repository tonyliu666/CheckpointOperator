package main

import (
	"flag"
	"fmt"
	"grpc-test/generated"
	"grpc-test/utility"
	"log"
	"net"

	"google.golang.org/grpc"
)

func main() {
	port := flag.Int("port", 50001, "the server port")
	flag.Parse()
	log.Printf("start server on port %d", *port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	generated.RegisterAgentServer(s, &utility.AgentServer{})

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
