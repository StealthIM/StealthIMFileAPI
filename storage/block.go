package storage

import (
	pb "StealthIMFileAPI/StealthIM.FileStorage"
	"StealthIMFileAPI/config"
	"context"
	"errors"
	"sync"
	"time"
)

var choosePointer = 0
var chooseLock = sync.Mutex{}

func chooseStorage() *storageConn {
	var conn *storageConn
	for range len(*Conns) {
		chooseLock.Lock()
		if choosePointer >= len(*Conns) {
			choosePointer = 0
		}
		conn = (*Conns)[choosePointer]
		choosePointer++
		chooseLock.Unlock()
		if conn == nil {
			continue
		}
		if conn.Conn == nil {
			continue
		}
		if conn.Usage >= conn.Total {
			continue
		}
		break
	}
	return conn
}

// SaveBytes 保存字节数据到存储
func SaveBytes(hash string, blockID int32, data []byte) (int32, error) {
	var latestCancelFunc func() = nil
	defer func() {
		if latestCancelFunc != nil {
			latestCancelFunc()
		}
	}()
	if Conns == nil {
		return 0, errors.New("No available storage connection")
	}
	for range len(*Conns) {
		if latestCancelFunc != nil {
			latestCancelFunc()
		}
		connObj := chooseStorage()
		if connObj == nil {
			continue
		}
		cli := pb.NewStealthIMFileStorageClient(connObj.Conn)
		if cli == nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.LatestConfig.Storage.Timeout)*time.Millisecond)
		latestCancelFunc = cancel
		if ctx == nil {
			continue
		}
		if connObj.Usage >= connObj.Total {
			continue
		}
		ret, err := cli.SaveFile(ctx, &pb.SaveFileRequest{
			Hash:      hash,
			Block:     blockID,
			BlockData: data,
		})
		if err != nil {
			continue
		}
		if ret.Result.Code != 0 {
			continue
		}
		connObj.Usage++
		return int32(connObj.ConnID), nil
	}
	return 0, errors.New("No available storage connection")
}

// GetBytes 从存储中获取字节数据
func GetBytes(hash string, blockID int32, connID int32) ([]byte, error) {
	conn := getConnFromID(int(connID))
	if conn == nil {
		return nil, errors.New("No such storage connection")
	}
	for range 3 {
		cli := pb.NewStealthIMFileStorageClient(conn.Conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.LatestConfig.Storage.Timeout)*time.Millisecond)
		ret, err := cli.GetFile(ctx, &pb.GetFileRequest{
			Hash:  hash,
			Block: blockID,
		})
		cancel()
		if err != nil {
			continue
		}
		return ret.BlockData, nil
	}
	return nil, errors.New("Get File network error")
}
