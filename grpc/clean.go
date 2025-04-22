package grpc

import (
	pb "StealthIMFileAPI/StealthIM.FileStorage"
	"StealthIMFileAPI/storage"
	"container/list"
	"context"
	"sync"
	"time"
)

var cleanChan = make(chan string, 64)
var cleanList = list.New()
var cleanMu sync.Mutex // 互斥锁保护 cleanList

func clean(uuid string) {
	for _, conn := range *storage.Conns {
		if conn.Conn == nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_, err := pb.NewStealthIMFileStorageClient(conn.Conn).RemoveBlock(ctx, &pb.RemoveBlockRequest{BlockHash: uuid})
		if err != nil {
			cancel()
			continue
		}
		cancel()
	}
}

// CleanTask 启动清理协程
func CleanTask() {
	updated := make(chan bool)
	go func() {
		for {
			cleanList.PushBack(<-cleanChan)
			cleanMu.Lock()
			if cleanList.Len() == 1 {
				updated <- true
			}
			cleanMu.Lock()
		}
	}()
	for {
		<-updated
		cleanMu.Lock()
		if cleanList.Len() > 0 {
			// clean操作
			clean(cleanList.Front().Value.(string))
			cleanList.Remove(cleanList.Front())
		}
		cleanMu.Unlock()

	}
}

func init() {
	go CleanTask()
}
