package storage

import (
	pb "StealthIMFileAPI/StealthIM.FileStorage"
	"StealthIMFileAPI/config"
	"StealthIMFileAPI/reload"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type storageConn struct {
	ConnID int
	Conn   *grpc.ClientConn
	Usage  int
	Total  int
	Host   string
	Port   int
}

// Conns 存储连接列表
var Conns *([]*storageConn)
var id2Conn *(map[int]int)
var mainlock sync.RWMutex

// SyncConns 启动定时同步状态任务
func SyncConns() {
	for {
		mainlock.RLock()
		if Conns == nil {
			continue
		}
		for nowConnID, conn := range *Conns {
			if conn.Conn == nil {
				conn.Usage = 0
				conn.Total = 0
				log.Printf("[Storage]Connect %d", nowConnID)
				cliconn, err := grpc.NewClient(fmt.Sprintf("%s:%d", conn.Host, conn.Port),
					grpc.WithTransportCredentials(
						insecure.NewCredentials()))
				if cliconn == nil {
					log.Printf("[Storage]Connect %d Error %v\n", nowConnID, err)
					conn.Conn = nil
					continue
				}
				if err != nil {
					log.Printf("[Storage]Connect %d Error %v\n", nowConnID, err)
					conn.Conn = nil
					continue
				}
				conn.Conn = cliconn
			}
			cli := pb.NewStealthIMFileStorageClient(conn.Conn)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			usageinfo, err := cli.GetUsage(ctx, &pb.GetUsageRequest{})
			if err != nil {
				conn.Conn = nil
				conn.Usage = 0
				conn.Total = 0
				log.Printf("[Storage]Sync %d Error %v\n", nowConnID, err)
			}
			if usageinfo != nil {
				conn.Usage = int(usageinfo.Usage)
				conn.Total = int(usageinfo.Total)
			}
			cancel()
		}
		mainlock.RUnlock()
		time.Sleep(time.Duration(config.LatestConfig.Storage.CheckTime) * time.Second)
	}
}

// LoadConnsFromConfig 从配置文件加载连接
func LoadConnsFromConfig() {
	mainlock.Lock()
	defer mainlock.Unlock()
	conntmp := make([]*storageConn, len(config.LatestConfig.FileStorage))
	id2ConnTmp := make(map[int]int, len(config.LatestConfig.FileStorage))
	for connNum, connCfg := range config.LatestConfig.FileStorage {
		conntmp[connNum] = &storageConn{ConnID: connCfg.ID, Conn: nil, Usage: 0, Total: 0, Host: connCfg.Host, Port: connCfg.Port}
		id2ConnTmp[connCfg.ID] = connNum
	}
	Conns = &conntmp
	id2Conn = &id2ConnTmp
}

// Start 启动链接
func Start() {
	go SyncConns()
	LoadConnsFromConfig()
	reload.ReloadCallback = append(reload.ReloadCallback, LoadConnsFromConfig)
}

func getConnFromID(connID int) *storageConn {
	mainlock.RLock()
	defer mainlock.RUnlock()
	connNum, ok := (*id2Conn)[connID]
	if !ok {
		return nil
	}
	return (*Conns)[connNum]
}
