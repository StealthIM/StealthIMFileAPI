PROTOCCMD = protoc
PROTOGEN_PATH = $(shell which protoc-gen-go) 
PROTOGENGRPC_PATH = $(shell which protoc-gen-go-grpc) 

GO_FILES := $(shell find $(SRC_DIR) -name '*.go')

GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean

LDFLAGS := -s -w

ifeq ($(OS), Windows_NT)
	DEFAULT_BUILD_FILENAME := StealthIMFileAPI.exe
else
	DEFAULT_BUILD_FILENAME := StealthIMFileAPI
endif

run: build
	./bin/$(DEFAULT_BUILD_FILENAME)

StealthIM.FileAPI/fileapi_grpc.pb.go StealthIM.FileAPI/fileapi.pb.go: proto/fileapi.proto
	$(PROTOCCMD) --plugin=protoc-gen-go=$(PROTOGEN_PATH) --plugin=protoc-gen-go-grpc=$(PROTOGENGRPC_PATH) --go-grpc_out=. --go_out=. proto/fileapi.proto

StealthIM.FileStorage/filestorage_grpc.pb.go StealthIM.FileStorage/filestorage.pb.go: proto/filestorage.proto
	$(PROTOCCMD) --plugin=protoc-gen-go=$(PROTOGEN_PATH) --plugin=protoc-gen-go-grpc=$(PROTOGENGRPC_PATH) --go-grpc_out=. --go_out=. proto/filestorage.proto

StealthIM.DBGateway/db_gateway_grpc.pb.go StealthIM.DBGateway/db_gateway.pb.go: proto/db_gateway.proto
	$(PROTOCCMD) --plugin=protoc-gen-go=$(PROTOGEN_PATH) --plugin=protoc-gen-go-grpc=$(PROTOGENGRPC_PATH) --go-grpc_out=. --go_out=. proto/db_gateway.proto

proto: ./StealthIM.FileAPI/fileapi_grpc.pb.go ./StealthIM.FileAPI/fileapi.pb.go StealthIM.FileStorage/filestorage_grpc.pb.go StealthIM.FileStorage/filestorage.pb.go ./StealthIM.DBGateway/db_gateway_grpc.pb.go ./StealthIM.DBGateway/db_gateway.pb.go


build: ./bin/$(DEFAULT_BUILD_FILENAME)

./bin/StealthIMFileAPI.exe: $(GO_FILES) proto
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o ./bin/StealthIMFileAPI.exe

./bin/StealthIMFileAPI: $(GO_FILES) proto
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ./bin/StealthIMFileAPI

build_win: ./bin/StealthIMFileAPI.exe
build_linux: ./bin/StealthIMFileAPI

# docker_run:
# 	docker-compose up

# ./bin/StealthIMFileAPI.docker.zst: $(GO_FILES) proto
# 	docker-compose build
# 	docker save stealthimfileapi-app > ./bin/StealthIMFileAPI.docker
# 	zstd ./bin/StealthIMFileAPI.docker -19
# 	@rm ./bin/StealthIMFileAPI.docker

# build_docker: ./bin/StealthIMFileAPI.docker.zst

# release: build_win build_linux build_docker

clean:
	@rm -rf ./StealthIM.FileStorage
	@rm -rf ./StealthIM.FileAPI
	@rm -rf ./bin
	@rm -rf ./__debug*

dev:
	./run_env.sh

debug_proto:
	cd test && python -m grpc_tools.protoc -I. --python_out=. --mypy_out=.  --grpclib_python_out=. --proto_path=../proto fileapi.proto

./tool/stimfileapi/proto/fileapi_grpc.py ./tool/stimfileapi/proto/fileapi_pb2.py ./tool/stimfileapi/proto/fileapi_pb2.pyi: proto/fileapi.proto
	@mkdir -p tool/stimfileapi/proto
	python -m grpc_tools.protoc -Iproto --python_out=./tool/stimfileapi/proto --grpclib_python_out=./tool/stimfileapi/proto --mypy_out=./tool/stimfileapi/proto proto/fileapi.proto
	@echo "Rewrite File"
	@sed -i 's/import fileapi_pb2/from . import fileapi_pb2/g' ./tool/stimfileapi/proto/fileapi_grpc.py

proto_t: ./tool/stimfileapi/proto/fileapi_grpc.py ./tool/stimfileapi/proto/fileapi_pb2.py ./tool/stimfileapi/proto/fileapi_pb2.pyi
