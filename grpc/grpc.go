package grpc

import (
	"StealthIMFileAPI/config"
	"StealthIMFileAPI/errorcode"
	"StealthIMFileAPI/reload"
	"context"
	"log"
	"net"
	"strconv"

	pb "StealthIMFileAPI/StealthIM.FileAPI"

	"google.golang.org/grpc"
)

type server struct {
	pb.StealthIMFileAPIServer
}

func (s *server) Ping(ctx context.Context, in *pb.PingRequest) (*pb.Pong, error) {
	return &pb.Pong{}, nil
}

func (s *server) Reload(ctx context.Context, in *pb.ReloadRequest) (*pb.ReloadResponse, error) {
	log.Println("[CONF]Reload config")
	go config.ReadConf()
	for _, f := range reload.ReloadCallback {
		go f()
	}
	return &pb.ReloadResponse{Result: &pb.Result{Code: errorcode.Success, Msg: ""}}, nil
}

// Start 启动 GRPC 服务
func Start(cfg config.Config) {
	lis, err := net.Listen("tcp", cfg.FileAPI.Host+":"+strconv.Itoa(cfg.FileAPI.Port))
	if err != nil {
		log.Fatalf("[GRPC]Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterStealthIMFileAPIServer(s, &server{})
	log.Printf("[GRPC]Server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("[GRPC]Failed to serve: %v", err)
	}
}
