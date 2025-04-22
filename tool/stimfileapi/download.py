from ast import Assert
from grpclib.client import Channel
from grpclib.exceptions import StreamTerminatedError, ProtocolError
from .proto import fileapi_grpc, fileapi_pb2
from rich.progress import Progress
import os
from rich import print


def merge_val(data: list[int]) -> list[tuple[int, int]]:
    result = []
    cnt = 0
    idx = -1

    for i, k in enumerate(data):
        if k == 0:
            cnt += 1
            if (idx == -1):
                idx = i
        else:
            if (idx != -1):
                result.append((idx, cnt))
                idx = -1
    if (idx != -1):
        result.append((idx, cnt))
    return result


async def download(server: str, port: int, file_path: str, block_size: int = 2*1024*1024, fhash: str = ""):
    blocknum = 0
    with Progress() as progress:
        task = None
        filesize = 0
        blocklst = []
        finishcnt = 0
        try:
            async with Channel(server, port) as channel:
                stub = fileapi_grpc.StealthIMFileAPIStub(channel)
                fileinfo = await stub.GetFileInfo(fileapi_pb2.GetFileInfoRequest(hash=fhash))
                if (fileinfo.result.code != 0):
                    raise Exception("Download Fault: "+fileinfo.result.msg)
                print(f"[blue]File size: [bold]{fileinfo.size}[/bold][/blue]")
                filesize = fileinfo.size
                blocknum = filesize // block_size
                if (fileinfo.size % block_size != 0):
                    blocknum += 1
                task = progress.add_task(
                    f"Downloading...", total=blocknum)
                blocklst = [0 for _ in range(blocknum)]
                for i in os.listdir(file_path+".stimfstmp"):
                    try:
                        bnum = int(i)
                        assert bnum < blocknum
                    except ValueError:
                        os.remove(file_path+".stimfstmp"+os.sep+i)
                    except AssertionError:
                        os.remove(file_path+".stimfstmp"+os.sep+i)
                    else:
                        blocklst[bnum] = 1
                        finishcnt += 1
                print(f"[green]Jump {finishcnt} block(s)[/green]")
                progress.update(task, completed=finishcnt)
                shortblocklist = merge_val(blocklst)
                while 1:
                    if (len(shortblocklist) == 0):
                        break
                    blockn = shortblocklist.pop(0)
                    async with stub.Download.open() as stream:
                        st = blockn[0]*block_size
                        ed = (blockn[0]+blockn[1])*block_size
                        if (blockn[0]+blockn[1] == blocknum):
                            ed = 0
                        await stream.send_message(fileapi_pb2.DownloadRequest(hash=fhash, start=st, end=ed))
                        async for chunk in stream:
                            if (chunk.WhichOneof('data') == "file"):
                                with open(file_path+".stimfstmp"+os.sep+str(chunk.file.blockid), "wb") as f:
                                    f.write(chunk.file.file)
                                finishcnt += 1
                                progress.update(task, completed=finishcnt)
                            elif (chunk.WhichOneof('data') == "result"):
                                if (chunk.result.code != 0):
                                    raise Exception(
                                        "Download Fault: "+chunk.result.msg)
                                else:
                                    break
                        await stream.end()
                channel.close()
        except StreamTerminatedError:
            pass
        except ProtocolError:
            pass
        exit_flag = True
        if (task is not None):
            progress.update(task, completed=blocknum)
    return blocknum
