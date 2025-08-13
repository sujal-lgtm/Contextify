package grpcserver

import (
	"context"
	"net"

	"github.com/sirupsen/logrus"
	pb "github.com/sujal-lgtm/Contextify/backend/services/contextify/proto"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedContextifyServiceServer
}

func (s *server) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Reply: "Pong: " + req.Message}, nil
}

func StartGrpcServer(port string) func() {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logrus.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterContextifyServiceServer(grpcServer, &server{})

	go func() {
		logrus.Infof("gRPC server listening on :%s", port)
		if err := grpcServer.Serve(lis); err != nil {
			logrus.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	return func() {
		logrus.Info("Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}
}
