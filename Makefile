PROTOC_GEN_GO := $(shell go env GOPATH)/bin/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(shell go env GOPATH)/bin/protoc-gen-go-grpc

.PHONY: all clean build_control build_node

all: pkg/grpc build_control build_node

pkg/grpc:
ifeq ($(OS),Windows_NT)
	if not exist pkg\grpc mkdir pkg\grpc
else
	mkdir -p pkg/grpc
endif
	protoc --proto_path=. --proto_path=grpc --go_out=pkg/ --go_opt=paths=source_relative --go-grpc_out=pkg/ --go-grpc_opt=paths=source_relative grpc/*.proto

build_control:
ifeq ($(OS),Windows_NT)
	if not exist bin mkdir bin
	go build -o bin/control.exe cmd/control/main.go
else
	mkdir -p bin
	go build -o bin/control cmd/control/main.go
endif

build_node:
ifeq ($(OS),Windows_NT)
	go build -o bin/node.exe cmd/node/main.go
else
	go build -o bin/node cmd/node/main.go
endif

clean:
ifeq ($(OS),Windows_NT)
	if exist pkg\grpc rmdir /s /q pkg\grpc
	if exist bin\control.exe del /f /q bin\control.exe
	if exist bin\node.exe del /f /q bin\node.exe
else
	rm -rf pkg/grpc
	rm -f bin/control
	rm -f bin/node
endif