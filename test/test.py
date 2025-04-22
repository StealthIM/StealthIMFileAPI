import grpclib
import fileapi_pb2 as pb
import fileapi_grpc as grpc
import asyncio
import blake3

from grpclib.client import Channel

BLOCKSIZE = 2*1024*1024


def chunk_bytes(data, chunk_size):
    for i in range(0, len(data), chunk_size):
        yield data[i:i + chunk_size]


async def run():
    async with Channel('127.0.0.1', 50053) as channel:
        stub = grpc.StealthIMFileAPIStub(channel)
        print("start")
        # 读取文件内容
        with open('test_file.txt', 'rb') as f:
            file_content = f.read()
        print("read")

        hashcnt = b''
        for i in chunk_bytes(file_content, BLOCKSIZE):
            hashcnt += blake3.blake3(i).digest()
        hashs = blake3.blake3(hashcnt).hexdigest()
        print("hash: ", hashs)
        req_init = pb.UploadRequest(metadata=pb.Upload_FileMetaData(
            totalsize=len(file_content), upload_uid=1, hash=hashs))
        print("send init")
        async with stub.Upload.open() as stream:
            await stream.send_message(req_init)
            resp = await stream.recv_message()
            print(resp)
            blocksize = resp.meta.blocksize
            for c, i in enumerate(chunk_bytes(file_content, blocksize)):
                req_upload = pb.UploadRequest(
                    file=pb.Upload_FileBlock(blockid=c, file=i))
                await stream.send_message(req_upload)
                resp = await stream.recv_message()
                print(resp)
            respf = await stream.recv_message()
            print(respf)
            await stream.cancel()


if __name__ == '__main__':
    asyncio.run(run())
