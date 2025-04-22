import blake3


def getfilehash(file_path: str, block_size: int = 2*1024*1024):
    hashbytes = b''
    with open(file_path, "rb") as f:
        while True:
            chunk = f.read(block_size)
            if not chunk:
                break
            hashbytes += blake3.blake3(chunk).digest()
    return blake3.blake3(hashbytes).hexdigest()
