package grpc

import (
	pb_gateway "StealthIMFileAPI/StealthIM.DBGateway"
	pb "StealthIMFileAPI/StealthIM.FileAPI"
	"StealthIMFileAPI/gateway"
	"context"

	"google.golang.org/protobuf/proto"
)

func (s *server) GetFileInfo(ctx context.Context, req *pb.GetFileInfoRequest) (*pb.GetFileInfoResponse, error) {

	hashByte := []byte{}
	for { // 流程控制用，只执行一次
		// 检查hash
		gret, gerr := gateway.ExecRedisBGet(&pb_gateway.RedisGetBytesRequest{Key: "files:filehash:" + req.Hash}) // 查缓存
		if gerr != nil && gret.Result.Code == 0 && len(gret.Value) > 0 {
			hashByte = gret.Value
			break
		}
		params := []*pb_gateway.InterFaceType{
			{
				Response: &pb_gateway.InterFaceType_Str{Str: req.Hash},
			},
		}
		ret, errsql := gateway.ExecSQL(&pb_gateway.SqlRequest{ // 查询数据库
			Db:              pb_gateway.SqlDatabases_File,
			Commit:          false,
			GetRowCount:     false,
			GetLastInsertId: false,
			Sql:             "SELECT `blocks`, `delete` FROM `files` WHERE hash = ? LIMIT 1",
			Params:          params,
		})
		if errsql != nil {
			break
		}
		if len(ret.Data) > 0 {
			if ret.Data[0].Result[1].GetInt32() != 1 {
				hashByte = ret.Data[0].Result[0].GetBlob()
				go gateway.ExecRedisBSet(&pb_gateway.RedisSetBytesRequest{
					Key:   "files:filehash:" + req.Hash,
					Value: hashByte,
					Ttl:   3600,
				})
			} else {
				go gateway.ExecRedisBSet(&pb_gateway.RedisSetBytesRequest{
					Key:   "files:filehash:" + req.Hash,
					Value: []byte{},
					Ttl:   3600,
				})
			}
			break
		}
		break
	}
	if len(hashByte) == 0 {
		return &pb.GetFileInfoResponse{Result: &pb.Result{Code: 1, Msg: "file not found"}}, nil
	}
	filemeta := &pb.BlockStorage{}
	unmasherr := proto.Unmarshal(hashByte, filemeta)
	if unmasherr != nil {
		return &pb.GetFileInfoResponse{Result: &pb.Result{Code: 2, Msg: "server error"}}, nil
	}
	return &pb.GetFileInfoResponse{Result: &pb.Result{Code: 0, Msg: ""}, Size: filemeta.Filesize}, nil
}
