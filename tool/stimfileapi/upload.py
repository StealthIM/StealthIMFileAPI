from grpclib.client import Channel
from grpclib.exceptions import StreamTerminatedError, ProtocolError
from .proto import fileapi_grpc, fileapi_pb2
from rich.progress import Progress
import os
import aiofiles
import asyncio


async def upload(server: str, port: int, file_path: str, block_size: int = 2*1024*1024, fhash: str = ""):
    filesize = os.stat(file_path).st_size
    fileblocknum = filesize//block_size
    if (filesize % block_size != 0):
        fileblocknum += 1
    with Progress() as progress:
        task = progress.add_task(
            f"Uploading...", total=fileblocknum)
        try:
            async with Channel(server, port) as channel:
                stub = fileapi_grpc.StealthIMFileAPIStub(channel)
                finished_num = 0
                sendlist = [i for i in range(fileblocknum)]
                retrylist = [0 for i in range(fileblocknum)]

                async def getblock(block_num: int):
                    async with aiofiles.open(file_path, 'rb') as f:
                        await f.seek(block_num*block_size)
                        data = await f.read(block_size)
                    return data
                exit_flag = False
                async with stub.Upload.open() as stream:
                    await stream.send_message(fileapi_pb2.UploadRequest(metadata=fileapi_pb2.Upload_FileMetaData(
                        totalsize=filesize, upload_uid=-1, hash=fhash)))

                    async def send():
                        await asyncio.sleep(1)
                        while not exit_flag:
                            if (len(sendlist)) > 0:
                                block_num = sendlist.pop(0)
                                try:
                                    data = await getblock(block_num)
                                    await stream.send_message(fileapi_pb2.UploadRequest(file=fileapi_pb2.Upload_FileBlock(blockid=block_num, file=data)))
                                except Exception:
                                    retrylist[block_num] += 1
                                    sendlist.append(block_num)
                                    if (retrylist[block_num] >= 3):
                                        raise Exception(
                                            "Upload Fault on "+str(block_num))
                            await asyncio.sleep(0.1)
                    asyncio.create_task(send())
                    while True:
                        response = await stream.recv_message()
                        if (response is None):
                            raise Exception("Connection Lost")
                        if (response.block != None):
                            if (response.result.code != 0):
                                sendlist.append(response.block.blockid)
                                retrylist[response.block.blockid] += 1
                                if (retrylist[response.block.blockid] >= 3):
                                    raise Exception(
                                        "Upload Fault: "+response.result.msg+" on "+str(response.block.blockid))
                                continue
                            finished_num += 1
                            progress.update(task, completed=finished_num)
                        elif (response.complete != None):
                            if (response.result.code != 0):
                                raise Exception(response.result.msg)
                            finished_num = fileblocknum
                            break
                    await stream.end()
        except StreamTerminatedError:
            pass
        except ProtocolError:
            pass
        exit_flag = True
        progress.update(task, completed=fileblocknum)
