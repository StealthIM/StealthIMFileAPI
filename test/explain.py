import fileapi_pb2

# 读取二进制文件
with open('storage_nodes.dat', 'rb') as f:
    data = f.read()

# 解析 StorageNodes
storage_nodes = fileapi_pb2.StorageNodes()
storage_nodes.ParseFromString(data)

print(str(storage_nodes))
