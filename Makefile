.PHONY: build run serve clean test tidy

BINARY=otter-ppt
BIN_DIR=bin

## build: 编译二进制
build:
	go build -o $(BIN_DIR)/$(BINARY) ./cmd/otter-ppt

## run: 直接运行
run:
	go run ./cmd/otter-ppt serve

## serve: 启动 HTTP 服务
serve: build
	$(BIN_DIR)/$(BINARY) serve

## clean: 清理构建产物
clean:
	rm -rf $(BIN_DIR) dist

## test: 运行测试
test:
	go test -v ./...

## tidy: 整理依赖
tidy:
	go mod tidy
