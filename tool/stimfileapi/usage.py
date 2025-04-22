from .proto import fileapi_grpc, fileapi_pb2
from grpclib.client import Channel
from rich import print


async def reload(host: str, port: int):
    channel = Channel(host, port)
    stub = fileapi_grpc.StealthIMFileAPIStub(channel)
    request = fileapi_pb2.ReloadRequest()
    response = await stub.Reload(request)
    print(repr(response))
    channel.close()


async def usage(host: str, port: int):
    channel = Channel(host, port)
    stub = fileapi_grpc.StealthIMFileAPIStub(channel)
    request = fileapi_pb2.UsageRequest()
    response = await stub.Usage(request)
    print(response)
    channel.close()
