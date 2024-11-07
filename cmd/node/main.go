package main

import (
	pb "galaxy/pkg/grpc"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"time"
)

type NodeServiceServer struct {
	pb.UnimplementedNodeServiceServer
}

// Heartbeat implements the heartbeat rpc method
func (s *NodeServiceServer) Heartbeat(ctx context.Context, in *emptypb.Empty) (*pb.HeartbeatResponse, error) {
	return &pb.HeartbeatResponse{Status: pb.HeartbeatResponse_UP}, nil
}

// StartGRPCServer starts the gRPC server.
func StartGRPCServer(port uint64) {
	lis, err := net.Listen("tcp", ":"+strconv.FormatUint(port, 10))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterNodeServiceServer(s, &NodeServiceServer{})

	// Register reflection service on gRPC server.
	reflection.Register(s)

	log.Println("gRPC server is running on port 50052")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	portArg := os.Args[1]
	port, err := strconv.ParseUint(portArg, 10, 32)
	if err != nil {
		log.Fatalf("could not parse port "+portArg+" with error: %v", err)
	}

	nodeSelf := pb.Node{
		ServiceName:    "node",
		ServiceAddress: "localhost",
		ServicePort:    uint32(port), // TODO change to uint64 in service
	}

	go StartGRPCServer(port)

	// Connect to the control service and register
	insecureCred := insecure.NewCredentials()
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecureCred))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("could not close connection: %v", err)
		}
	}(conn)

	client := pb.NewControlServiceClient(conn)
	_, err = client.Heartbeat(context.Background(), &emptypb.Empty{})
	if err != nil {
		log.Fatalf("could not contact control service: %v", err)
	}

	_, err = client.Register(context.Background(), &nodeSelf)

	if err != nil {
		log.Fatalf("could not register: %v", err)
		return
	}

	defer func() {
		_, err = client.Deregister(context.Background(), &nodeSelf)
		if err != nil {
			log.Fatalf("could not deregister: %v", err)
		}
	}()

	// Handle SIGINT
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			_, err = client.Deregister(context.Background(), &nodeSelf)
			if err != nil {
				log.Fatalf("could not deregister: %v", err)
			}

			// sig is a ^C, handle it
			os.Exit(0)
		}
	}()

	// Keep the main function running
	for true {
		nodes, err := client.ListNodes(context.Background(), &nodeSelf)
		if err != nil {
			log.Fatalf("could not list nodes: %v", err)
			return
		}

		// clear terminal and print nodes
		print("\033[H\033[2J")
		log.Println("Nodes:")
		for {
			node, err := nodes.Recv()
			if err != nil {
				break
			}
			log.Println(node)
		}

		// Sleep for 5 seconds
		<-time.After(5 * time.Second)
	}
}
