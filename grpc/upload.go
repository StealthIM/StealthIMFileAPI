package grpc

import (
	pb_gateway "StealthIMFileAPI/StealthIM.DBGateway"
	pb "StealthIMFileAPI/StealthIM.FileAPI"
	pb_msap "StealthIMFileAPI/StealthIM.MSAP"
	"StealthIMFileAPI/config"
	"StealthIMFileAPI/errorcode"
	"StealthIMFileAPI/gateway"
	"StealthIMFileAPI/msap"
	"StealthIMFileAPI/storage"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/google/uuid"
	"github.com/zeebo/blake3"
)

func (s *server) Upload(stream pb.StealthIMFileAPI_UploadServer) error {
	cfgCopy := config.LatestConfig
	if cfgCopy.FileAPI.Log {
		log.Printf("[FileAPI] Call upload")
	}
	var uploadMode = 0
	var filemeta *pb.Upload_FileMetaData
	var blocksize = cfgCopy.Storage.BlockSize

	// 接受元数据
	in, err := stream.Recv()
	if err != nil {
		return err
	}
	if metainfo := in.GetMetadata(); metainfo != nil {
		// 正确的元数据
		filemeta = metainfo
	} else {
		// 空元数据
		if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.FSAPMetadataEmpty, Msg: "metadata is empty"}, Data: &pb.UploadResponse_Meta{Meta: &pb.Upload_MetaResponse{Blocksize: (int32)(blocksize) * 1024}}}); err != nil {
			return err
		}
		handleStream(stream) // 等待流结束
		return nil
	}

	// 发送元数据反馈
	if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.Success, Msg: ""}, Data: &pb.UploadResponse_Meta{Meta: &pb.Upload_MetaResponse{Blocksize: (int32)(blocksize) * 1024}}}); err != nil {
		return err
	}

	if len(filemeta.Hash) != 64 {
		if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.FSAPHashBroken, Msg: "hash is broken"}, Data: &pb.UploadResponse_Meta{Meta: &pb.Upload_MetaResponse{Blocksize: (int32)(blocksize) * 1024}}}); err != nil {
			return err
		}
		handleStream(stream) // 等待流结束
		return nil
	}
	if filemeta.Totalsize == 0 {
		if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.FSAPFileEmpty, Msg: "file is empty"}, Data: &pb.UploadResponse_Meta{Meta: &pb.Upload_MetaResponse{Blocksize: (int32)(blocksize) * 1024}}}); err != nil {
			return err
		}
		handleStream(stream) // 等待流结束
		return nil
	}
	if filemeta.UploadUid == 0 {
		if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.FSAPMetadataError, Msg: "UID is empty"}, Data: &pb.UploadResponse_Meta{Meta: &pb.Upload_MetaResponse{Blocksize: (int32)(blocksize) * 1024}}}); err != nil {
			return err
		}
		handleStream(stream) // 等待流结束
		return nil
	}
	if filemeta.UploadGroupid == 0 {
		if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.FSAPMetadataError, Msg: "GroupID is empty"}, Data: &pb.UploadResponse_Meta{Meta: &pb.Upload_MetaResponse{Blocksize: (int32)(blocksize) * 1024}}}); err != nil {
			return err
		}
		handleStream(stream) // 等待流结束
		return nil
	}

	callbackMsg := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		filename := filemeta.Filename
		if len(filename) == 0 {
			filename = fmt.Sprintf("Unknown.%s.txt", filemeta.Hash[:8])
		}
		var err error
		var resp *pb_msap.FileAPICallResponse
		if cfgCopy.FileAPI.Log {
			log.Printf("[FileAPI] Callback")
		}
		for range 3 {
			resp, err = msap.FileAPICall(ctx, &pb_msap.FileAPICallRequest{
				Uid:      filemeta.UploadUid,
				Groupid:  filemeta.UploadGroupid,
				Hash:     filemeta.Hash,
				Filename: filemeta.Filename,
			})
			if err == nil {
				if resp.Result.Code == errorcode.Success {
					return
				}
			}
			time.Sleep(3 * time.Second)
		}
		if err != nil {
			log.Printf("[FileAPI] Call callback error: %v", err)
		}

	}

	// 检查hash
	gret, gerr := gateway.ExecRedisBGet(&pb_gateway.RedisGetBytesRequest{Key: "files:filehash:" + filemeta.Hash}) // 查缓存
	if gerr != nil && gret != nil && gret.Result != nil && gret.Result.Code == errorcode.Success {
		if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.Success, Msg: ""}, Data: &pb.UploadResponse_Complete{Complete: &pb.Upload_CompleteResponse{Hash: filemeta.Hash}}}); err != nil {
			return errors.New("Return error")
		}

		if config.LatestConfig.Callback.Host != "" {
			go callbackMsg()
		}

		handleStream(stream) // 等待流
		return nil
	}
	params := []*pb_gateway.InterFaceType{
		{
			Response: &pb_gateway.InterFaceType_Str{Str: filemeta.Hash},
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
		return errsql
	}
	if len(ret.Data) > 0 {
		if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.Success, Msg: ""}, Data: &pb.UploadResponse_Complete{Complete: &pb.Upload_CompleteResponse{Hash: filemeta.Hash}}}); err != nil {
			return errors.New("Return error")
		}

		if ret.Data[0].Result[1].GetInt32() != 1 {
			go gateway.ExecRedisBSet(&pb_gateway.RedisSetBytesRequest{
				Key:   "files:filehash:" + filemeta.Hash,
				Value: ret.Data[0].Result[0].GetBlob(),
				Ttl:   3600,
			})
		} else {
			go gateway.ExecRedisBSet(&pb_gateway.RedisSetBytesRequest{
				Key:   "files:filehash:" + filemeta.Hash,
				Value: []byte{},
				Ttl:   3600,
			})
		}

		if config.LatestConfig.Callback.Host != "" {
			go callbackMsg()
		}

		handleStream(stream) // 等待流
		return nil
	}

	// 计算数据
	var blocknum = int32(filemeta.Totalsize / ((int64)(blocksize) * 1024))
	var lastBlockSize = (int)(blocksize) * 1024
	if filemeta.Totalsize%((int64)(blocksize)*1024) != 0 {
		lastBlockSize = (int)(filemeta.Totalsize) % ((int)(blocksize) * 1024)
		blocknum++
	}
	var blocksInfo = make([]int32, blocknum) // blocks 存储ID
	var successBlockCnt int32 = 0            // 成功块数量
	var blockHash = make([][]byte, blocknum) // block hash
	for i := range blocksInfo {              // 初始化 ID
		blocksInfo[i] = -1
	}

	if blocknum == 1 {
		uploadMode = 1 // 小块存储模式
	}
	var blockDataTmp *[]byte

	var uploadEnable = true             // 上传标志
	var closeChannel = make(chan error) // 关闭 flag

	defer func() { uploadEnable = false }() // 关闭标志

	var fileID string = uuid.New().String() // 初始化文件名

	var cleanFlag = true
	defer func() {
		if cleanFlag {
			cleanChan <- fileID
		}
	}()

	go func() {
		for {
			in, err := stream.Recv()
			if err != nil {
				closeChannel <- err // 关闭 goro
				return
			}
			if metainfo := in.GetMetadata(); metainfo != nil {
				// 处理元数据重复
				if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.FSAPMetadataError, Msg: "metadata is repeated"}, Data: &pb.UploadResponse_Meta{Meta: &pb.Upload_MetaResponse{Blocksize: (int32)(blocksize) * 1024}}}); err != nil {
					closeChannel <- err
					return
				}
				closeChannel <- errors.New("metadata is repeated")
				return
			} else if blockinfo := in.GetFile(); blockinfo != nil {
				// 文件块超出范围
				if blockinfo.Blockid < 0 || blockinfo.Blockid >= blocknum {
					if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.FSAPMetadataError, Msg: "blocknum out of range"}, Data: &pb.UploadResponse_Meta{Meta: &pb.Upload_MetaResponse{Blocksize: (int32)(blocksize) * 1024}}}); err != nil {
						closeChannel <- err
						return
					}
					closeChannel <- errors.New("blocknum out of range")
					return
				}
				// 文件块重复
				if blocksInfo[blockinfo.Blockid] != -1 {
					if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.FSAPUploadBlockRepeat, Msg: "blockid repeated"}, Data: &pb.UploadResponse_Meta{Meta: &pb.Upload_MetaResponse{Blocksize: (int32)(blocksize) * 1024}}}); err != nil {
						closeChannel <- err
						return
					}
					closeChannel <- nil
					return
				}
				// 完整文件块大小错误
				if blockinfo.Blockid < blocknum-1 && len(blockinfo.File) != blocksize*1024 {
					if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.FSAPMetadataError, Msg: "block size wrong"}, Data: &pb.UploadResponse_Meta{Meta: &pb.Upload_MetaResponse{Blocksize: (int32)(blocksize) * 1024}}}); err != nil {
						closeChannel <- err
						return
					}
					closeChannel <- errors.New("block size wrong")
					return
				} else if blockinfo.Blockid == blocknum-1 && len(blockinfo.File) != lastBlockSize {
					// 尾文件块大小错误
					if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.FSAPMetadataError, Msg: "block size wrong"}, Data: &pb.UploadResponse_Meta{Meta: &pb.Upload_MetaResponse{Blocksize: (int32)(blocksize) * 1024}}}); err != nil {

						closeChannel <- err
						return
					}
					closeChannel <- errors.New("block size wrong")
					return
				}
				// 启动upload任务
				upload := func(blockinfo *pb.Upload_FileBlock) {
					if !uploadEnable {
						return
					}
					if uploadMode == 1 {
						blockDataTmp = &blockinfo.File

						// 发送反馈
						if err2 := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.Success, Msg: ""}, Data: &pb.UploadResponse_Block{Block: &pb.Upload_BlockResponse{Blockid: 0}}}); err2 != nil {
							return
						}
						closeChannel <- nil
						return
					}
					// 存储任务
					blockid, err := storage.SaveBytes(fileID, blockinfo.Blockid, blockinfo.File)
					if !uploadEnable {
						return
					}
					if err != nil { // 存储失败
						if err2 := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.ServerInternalComponentError, Msg: err.Error()}, Data: &pb.UploadResponse_Block{Block: &pb.Upload_BlockResponse{Blockid: blockinfo.Blockid}}}); err2 != nil {
							closeChannel <- errors.New("send error")
							return
						}
						return
					}
					// 存储成功
					hasher := blake3.New()
					hasher.Write(blockinfo.File)
					blockHash[blockinfo.Blockid] = hasher.Sum(nil)
					blocksInfo[blockinfo.Blockid] = blockid
					successBlockCnt++
					// 发送反馈
					if err2 := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.Success, Msg: ""}, Data: &pb.UploadResponse_Block{Block: &pb.Upload_BlockResponse{Blockid: blockinfo.Blockid}}}); err2 != nil {
						return
					}
					if successBlockCnt == blocknum {
						closeChannel <- nil
						return
					}
				}
				go upload(blockinfo)
			} else { // 错误的block
				if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.FSAPMetadataError, Msg: "blockinfo is empty"}, Data: &pb.UploadResponse_Meta{Meta: &pb.Upload_MetaResponse{Blocksize: (int32)(blocksize) * 1024}}}); err != nil {
					closeChannel <- err
					return
				}
				closeChannel <- errors.New("blockinfo is empty")
				return
			}
		}
	}()

	// 等待关闭
	closeErr := <-closeChannel
	if closeErr != nil {
		return closeErr
	}

	// 准备存储对象
	var blobdata []byte
	var berr error
	if uploadMode == 0 { // 默认模式
		blobdata, berr = proto.Marshal(&pb.BlockStorage{Filename: fileID, Filesize: filemeta.Totalsize, Type: &pb.BlockStorage_Nodes{Nodes: &pb.StorageNodes{Nodeid: blocksInfo}}})
	} else { // 小文件
		hasher := blake3.New()
		hasher.Write(*blockDataTmp)
		blockHash[0] = hasher.Sum(nil)
		blobdata, berr = proto.Marshal(&pb.BlockStorage{Filename: fileID, Filesize: filemeta.Totalsize, Type: &pb.BlockStorage_Data{Data: *blockDataTmp}})
	}
	if berr != nil {
		if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.FSAPSaveBlockStorageFault, Msg: "save blockstorage fault"}, Data: &pb.UploadResponse_Complete{Complete: &pb.Upload_CompleteResponse{}}}); err != nil {
			return errors.New("save blockstorage fault")
		}
		return berr
	}

	// 计算文件hash
	hasher := blake3.New()
	for _, nowhash := range blockHash {
		hasher.Write(nowhash)
	}
	fileFullHash := hex.EncodeToString(hasher.Sum(nil))

	// 写入数据库
	params = []*pb_gateway.InterFaceType{
		{
			Response: &pb_gateway.InterFaceType_Str{Str: fileFullHash},
		},
		{
			Response: &pb_gateway.InterFaceType_Blob{Blob: blobdata},
		},
		{
			Response: &pb_gateway.InterFaceType_Int64{Int64: filemeta.Totalsize},
		},
		{
			Response: &pb_gateway.InterFaceType_Int32{Int32: filemeta.UploadUid},
		},
	}
	ret, err2 := gateway.ExecSQL(&pb_gateway.SqlRequest{
		Db:              pb_gateway.SqlDatabases_File,
		Commit:          true,
		GetRowCount:     false,
		GetLastInsertId: false,
		Sql:             "INSERT INTO `files` (`hash`, `blocks`, `filesize`, `upload_uid`) VALUES (?, ?, ?, ?)",
		Params:          params,
	})

	// 检查hash
	if filemeta.Hash != "" && filemeta.Hash != fileFullHash {
		if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.FSAPHashNotMatch, Msg: "hash not match"}, Data: &pb.UploadResponse_Complete{Complete: &pb.Upload_CompleteResponse{}}}); err != nil {
			return errors.New("hash not match")
		}
		handleStream(stream) // 等待流
		return nil
	}

	if err2 != nil || ret.Result.Code != errorcode.Success {
		if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.FSAPUploadToDatabaseFault, Msg: "upload to database fault"}, Data: &pb.UploadResponse_Complete{Complete: &pb.Upload_CompleteResponse{}}}); err != nil {
			return errors.New("upload to database fault")
		}
		handleStream(stream) // 等待流
		return nil
	}

	// 返回
	if err := stream.Send(&pb.UploadResponse{Result: &pb.Result{Code: errorcode.Success, Msg: ""}, Data: &pb.UploadResponse_Complete{Complete: &pb.Upload_CompleteResponse{Hash: fileFullHash}}}); err != nil {
		return errors.New("Return error")
	}

	cleanFlag = false

	go gateway.ExecRedisDel(&pb_gateway.RedisDelRequest{
		DBID: 0,
		Key:  "files:filehash:" + fileFullHash,
	})

	if config.LatestConfig.Callback.Host != "" {
		go callbackMsg()
	}

	handleStream(stream) // 等待流
	return nil
}
