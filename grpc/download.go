package grpc

import (
	pb_gateway "StealthIMFileAPI/StealthIM.DBGateway"
	pb "StealthIMFileAPI/StealthIM.FileAPI"
	"StealthIMFileAPI/config"
	"StealthIMFileAPI/gateway"
	"StealthIMFileAPI/storage"
	"context"
	"errors"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"
)

func (s *server) Download(req *pb.DownloadRequest, stream pb.StealthIMFileAPI_DownloadServer) error {
	blocksize := config.LatestConfig.Storage.BlockSize * 1024

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
		if err := stream.Send(&pb.DownloadResponse{Data: &pb.DownloadResponse_Result{Result: &pb.Result{Code: 1, Msg: "file not found"}}}); err != nil {
			return errors.New("file not found")
		}
		time.Sleep(500 * time.Millisecond)
		return nil
	}
	filemeta := &pb.BlockStorage{}
	unmasherr := proto.Unmarshal(hashByte, filemeta)
	if unmasherr != nil {
		if err := stream.Send(&pb.DownloadResponse{Data: &pb.DownloadResponse_Result{Result: &pb.Result{Code: 2, Msg: "server unmarshal error"}}}); err != nil {
			return errors.New("server unmarshal error")
		}
		time.Sleep(500 * time.Millisecond)
		return nil
	}
	start := (int)(req.Start)
	end := (int)(req.End)
	if req.End == 0 {
		end = (int)(filemeta.Filesize)
	}
	if start >= end {
		if err := stream.Send(&pb.DownloadResponse{Data: &pb.DownloadResponse_Result{Result: &pb.Result{Code: 3, Msg: "start must less than end"}}}); err != nil {
			return errors.New("start must less than end")
		}
		time.Sleep(500 * time.Millisecond)
		return nil
	}
	if start < 0 || end > (int)(filemeta.Filesize)+1 {
		if err := stream.Send(&pb.DownloadResponse{Data: &pb.DownloadResponse_Result{Result: &pb.Result{Code: 4, Msg: "out of the range"}}}); err != nil {
			return errors.New("out of the range")
		}
		time.Sleep(500 * time.Millisecond)
		return nil
	}
	uuid := filemeta.Filename
	startBlock := (int)(start) / blocksize
	endBlock := (int)(end-1) / blocksize
	startOffset := start % blocksize
	endOffset := end % blocksize
	if end%blocksize == 0 {
		endOffset = blocksize
	}
	fd := filemeta.GetData()
	if fd != nil {
		if startOffset > 0 {
			fd = fd[startOffset:]
		}
		if endOffset < len(fd) {
			fd = fd[:endOffset]
		}
		if err := stream.Send(&pb.DownloadResponse{Data: &pb.DownloadResponse_File{File: &pb.Download_FileBlock{Blockid: 0, File: fd}}}); err != nil {
			return errors.New("send data fault")
		}
		if err := stream.Send(&pb.DownloadResponse{Data: &pb.DownloadResponse_Result{Result: &pb.Result{Code: 0, Msg: ""}}}); err != nil {
			return errors.New("send result fault")
		}
		time.Sleep(500 * time.Millisecond)
		return nil
	}
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	sendedNum := (int32)(startBlock)
	errchan := make(chan error, 1)
	blockLst := &filemeta.GetNodes().Nodeid
	sendBlock := func(block int, retry uint16) {}
	sendBlock = func(block int, retry uint16) {
		if retry > 3 {
			errchan <- errors.New("send block error")
			return
		} else if retry > 1 {
			time.Sleep(200 * time.Millisecond)
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		if block >= len(*blockLst) {
			errchan <- errors.New("send result fault")
			return
		}
		bytes, geterr := storage.GetBytes(uuid, (int32)(block), (*blockLst)[block])
		if geterr != nil {
			go sendBlock(block, retry+1)
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		if block == startBlock {
			bytes = bytes[startOffset:]
		}
		if block == endBlock && len(bytes) > endOffset {
			bytes = bytes[:endOffset]
		}
		if err := stream.Send(&pb.DownloadResponse{Data: &pb.DownloadResponse_File{File: &pb.Download_FileBlock{Blockid: int32(block), File: bytes}}}); err != nil {
			go sendBlock(block, retry+1)
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		atomic.AddInt32(&sendedNum, 1)
		if sendedNum > (int32)(endBlock) {
			time.Sleep(100 * time.Millisecond)
			if err := stream.Send(&pb.DownloadResponse{Data: &pb.DownloadResponse_Result{Result: &pb.Result{Code: 0, Msg: ""}}}); err != nil {
				errchan <- errors.New("send result fault")
				return
			}
			errchan <- nil
			return
		}
	}
	for i := startBlock; i <= endBlock; i++ {
		go sendBlock(i, 1)
		time.Sleep(50 * time.Millisecond)
	}
	ret := <-errchan
	cancel()
	time.Sleep(500 * time.Millisecond)
	return ret
}
