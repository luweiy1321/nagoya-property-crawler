.PHONY: all build run-server run-crawler test clean deps migrate

# 变量
BINARY_SERVER=bin/server
BINARY_CRAWLER=bin/crawler
GO=go
CONFIG_FILE=config.yaml

# 默认目标
all: deps build

# 安装依赖
deps:
	$(GO) mod download
	$(GO) mod tidy

# 构建
build: build-server build-crawler

build-server:
	mkdir -p bin
	$(GO) build -o $(BINARY_SERVER) cmd/server/main.go

build-crawler:
	mkdir -p bin
	$(GO) build -o $(BINARY_CRAWLER) cmd/crawler/main.go

# 运行服务器
run-server: deps
	$(GO) run cmd/server/main.go -config=$(CONFIG_FILE)

# 运行爬虫
run-crawler: deps
	$(GO) run cmd/crawler/main.go -config=$(CONFIG_FILE)

# 爬取单个来源
run-suumo: deps
	$(GO) run cmd/crawler/main.go -config=$(CONFIG_FILE) -source=suumo

run-homes: deps
	$(GO) run cmd/crawler/main.go -config=$(CONFIG_FILE) -source=homes

run-athome: deps
	$(GO) run cmd/crawler/main.go -config=$(CONFIG_FILE) -source=athome

# 测试
test:
	$(GO) test -v ./...

# 测试覆盖率
coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out

# 代码检查
lint:
	$(GO) vet ./...
	gofmt -l .

# 格式化代码
fmt:
	gofmt -w .

# 清理
clean:
	rm -rf bin/
	rm -f coverage.out

# 数据库迁移
migrate-init:
	psql -U postgres -c "CREATE DATABASE nagoya_properties;" || true
	psql -U postgres -d nagoya_properties -f migrations/001_init.sql

# 数据库清理
migrate-drop:
	psql -U postgres -c "DROP DATABASE IF EXISTS nagoya_properties;"

# 重置数据库
migrate-reset: migrate-drop migrate-init

# Docker 构建
docker-build:
	docker build -t nagoya-property-crawler .

# Docker 运行
docker-run:
	docker-compose up -d

# 开发环境
dev: deps migrate-init
	@echo "开发环境准备完成"
	@echo "运行 'make run-server' 启动服务器"
	@echo "运行 'make run-crawler' 启动爬虫"

# 帮助
help:
	@echo "可用命令:"
	@echo "  make deps           - 安装依赖"
	@echo "  make build          - 构建所有二进制文件"
	@echo "  make build-server   - 构建服务器"
	@echo "  make build-crawler  - 构建爬虫"
	@echo "  make run-server     - 运行服务器"
	@echo "  make run-crawler    - 运行爬虫"
	@echo "  make run-suumo      - 爬取 SUUMO"
	@echo "  make run-homes      - 爬取 HOMES"
	@echo "  make run-athome     - 爬取 at-home"
	@echo "  make test           - 运行测试"
	@echo "  make coverage       - 测试覆盖率"
	@echo "  make migrate-init   - 初始化数据库"
	@echo "  make migrate-reset  - 重置数据库"
	@echo "  make clean          - 清理构建文件"
