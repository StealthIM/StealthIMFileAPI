package grpc

import (
	pb "StealthIMFileAPI/StealthIM.FileAPI"
	"context"
	"time"
)

// 等待流结束
func handleStream(stream pb.StealthIMFileAPI_UploadServer) error {
	// 创建一个1秒的超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	for {
		// 创建一个接收消息的通道
		msgChan := make(chan *pb.UploadRequest, 1)

		// 在一个新的goroutine中接收消息
		go func() {
			var tmp *pb.UploadRequest
			stream.RecvMsg(&tmp)
			msgChan <- nil
		}()

		// 使用select等待消息接收完成或者上下文超时
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-msgChan:
			return nil
		}
	}
}
