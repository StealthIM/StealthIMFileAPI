import asyncio
from grpclib.client import Channel
import fileapi_pb2
import fileapi_grpc

async def get_usage():
    # 创建通道
    channel = Channel('localhost', 50053)

    # 创建客户端存根
    stub = fileapi_grpc.StealthIMFileAPIStub(channel)

    # 创建UsageRequest
    request = fileapi_pb2.UsageRequest()

    # 调用Usage RPC
    response = await stub.Usage(request)

    print(response)


# 运行异步函数
asyncio.run(get_usage())
