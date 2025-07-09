package grpc

import (
	pb "StealthIMFileAPI/StealthIM.FileAPI"
	"StealthIMFileAPI/config"
	"StealthIMFileAPI/errorcode"
	"StealthIMFileAPI/storage"
	"context"
	"log"
)

func (s *server) Usage(ctx context.Context, in *pb.UsageRequest) (*pb.UsageResponse, error) {
	if config.LatestConfig.FileAPI.Log {
		log.Println("[GRPC]Call Usage")
	}
	usageTmp := make([]*pb.UsageNode, 0)
	for _, node := range *storage.Conns {
		usageTmp = append(usageTmp, &pb.UsageNode{Usage: (int32)(node.Usage), Id: int32(node.ConnID), Total: (int32)(node.Total), Online: node.Conn != nil})
	}
	return &pb.UsageResponse{Nodes: usageTmp, Result: &pb.Result{Code: errorcode.Success, Msg: ""}}, nil
}

func (s *server) GetBlockSize(ctx context.Context, in *pb.GetBlockSizeRequest) (*pb.GetBlockSizeResponse, error) {
	if config.LatestConfig.FileAPI.Log {
		log.Println("[GRPC]Call GetBlockSize")
	}
	return &pb.GetBlockSizeResponse{Blocksize: int32(config.LatestConfig.Storage.BlockSize) * 1024}, nil
}
