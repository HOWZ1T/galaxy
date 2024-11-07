package main

import (
	pb "galaxy/pkg/grpc"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
)

type StateContainer struct {
	mu    sync.Mutex
	nodes map[string]*pb.Node
}

type ControlServiceServer struct {
	pb.UnimplementedControlServiceServer
	state StateContainer
}

func constructNodeKey(node *pb.Node) string {
	return node.ServiceAddress + ":" + strconv.FormatUint(uint64(node.ServicePort), 10) + "-" + strings.ToLower(node.ServiceName)
}

// Heartbeat implements the heartbeat rpc method
func (s *ControlServiceServer) Heartbeat(ctx context.Context, in *emptypb.Empty) (*pb.HeartbeatResponse, error) {
	return &pb.HeartbeatResponse{Status: pb.HeartbeatResponse_UP}, nil
}

func (s *ControlServiceServer) Register(ctx context.Context, in *pb.Node) (*pb.RegisterResponse, error) {
	key := constructNodeKey(in)

	// Acquire the lock
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	// Check if the service is already registered
	if _, ok := s.state.nodes[key]; ok {
		msg := "service with name " + key + " already registered"
		log.Println("register failed: " + msg)
		return &pb.RegisterResponse{
			Message: msg,
			Success: false,
		}, nil
	}

	// Register the service
	s.state.nodes[key] = in
	msg := "service with name " + key + " registered"
	log.Println(msg)
	return &pb.RegisterResponse{
		Message: msg,
		Success: true,
	}, nil
}

func (s *ControlServiceServer) Deregister(ctx context.Context, in *pb.Node) (*pb.RegisterResponse, error) {
	key := constructNodeKey(in)

	// Acquire the lock
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	// Check if the service is already registered
	if _, ok := s.state.nodes[key]; !ok {
		msg := "service with name " + key + " not found"
		log.Println("deregister failed: " + msg)
		return &pb.RegisterResponse{
			Message: msg,
			Success: false,
		}, nil
	}

	// Deregister the service
	delete(s.state.nodes, key)
	msg := "service with name " + key + " deregistered"
	log.Println(msg)
	return &pb.RegisterResponse{
		Message: msg,
		Success: true,
	}, nil
}

func (s *ControlServiceServer) ListNodes(in *pb.Node, srv grpc.ServerStreamingServer[pb.Node]) error {
	// TODO: does it need to be locked?
	keyRequestingNode := constructNodeKey(in)
	for _, node := range s.state.nodes {
		key := constructNodeKey(node)
		if key == keyRequestingNode {
			continue
		} // Omit the requesting node from the response

		if err := srv.Send(node); err != nil {
			return err
		}
	}
	return nil
}

// StartGRPCServer starts the gRPC server.
func StartGRPCServer() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(grpc.MaxConcurrentStreams(math.MaxUint32))
	pb.RegisterControlServiceServer(s, &ControlServiceServer{
		state: StateContainer{
			mu:    sync.Mutex{},
			nodes: make(map[string]*pb.Node),
		},
	})

	// Register reflection service on gRPC server.
	reflection.Register(s)

	log.Println("gRPC server is running on port 50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	StartGRPCServer()
}
