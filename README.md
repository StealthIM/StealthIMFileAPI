# StealthIM FileAPI

文件接口 `0.0.1`

> `.proto` 文件：`./proto/fileapi.proto`

## hash 计算

hash 算法采用分块 Blake3 进行。具体分块如下：

将文件以预设大小分块（默认为 2048 KiB，需要与服务器一致）后计算每块 Blake3，将计算后的 hash binary 形式拼接后再计算一次 Blake3，得到最终 hash。

## 构建

### 依赖

Go 版本：`1.24.1`

软件包：`protobuf` `protobuf-dev` `make`

> 命令行工具 `protoc` `make`(gnumake)

```bash
go mod download
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### 命令行

```bash
make # 构建并运行
make build # 构建可执行文件

# 构建指定环境
make build_windows
make build_linux
make build_docker

make release # 构建所有平台

make proto # 生成 proto

make clean # 清理
```

## 配置

默认会读取当前文件夹 `config.toml` 文件（不存在会自动生成模板）

> docker 镜像为 `cfg/config.toml`

默认配置：

```toml
[server]
host = "127.0.0.1"
port = 50053
log = false

[dbgateway]
host = "127.0.0.1"
port = 50051
conn_num = 5
sql_timeout = 5000 # 单位：ms

[[filestorage]]
host = "127.0.0.1"
port = 50052
id = 0

# 多个存储单元配置
#
# [[filestorage]]
# host = "192.168.xx.xx"
# port = 50052
# id = 1
#
# 请保证 ID 唯一，在迁移时请使用相同的 ID

[storage]
timeout = 1000   # 单位：ms
check_time = 60  # 单位：s
blocksize = 2048 # 单位：KB
# 不建议更改，不超过 2048
```

也可使用 `--config={PATH}` 参数指定配置文件路径

## 命令行工具

命令行工具使用 Typer 构建。

命令行工具在打包前需要使用 makefile 构建 proto。

```bash
make proto_t # 生成工具用 proto
make clean # 清理
```

使用 `poetry install` 安装。

帮助信息请查看 `stimfileapi --help`。

## 调试环境

确保你使用的是 Linux 环境并安装了 `tmux`。

确保在父目录以原名 Clone 了 `StealthimDB` 和 `StealthIMFileStorage`。

```bash
make dev
# 命令会运行 run_env.sh
```
