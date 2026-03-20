.PHONY: run-backend run-frontend test-report minio-up minio-down help

# 默认命令
help:
	@echo "Available commands:"
	@echo "  make run-backend   - Start the Go backend server"
	@echo "  make run-frontend  - Start the Vite frontend development server"
	@echo "  make test-report   - Run the coverage generation test (uint_cover_test.go)"
	@echo "  make minio-up      - Start MinIO OSS service"
	@echo "  make minio-down    - Stop MinIO OSS service"

# 启动 MinIO 服务
minio-up:
	@echo "Starting MinIO service..."
	docker-compose up -d

# 停止 MinIO 服务
minio-down:
	@echo "Stopping MinIO service..."
	docker-compose down

# 启动后端服务器
run-backend:
	@echo "Starting backend server..."
	go run cmd/cover-server/main.go

# 启动前端服务器
run-frontend:
	@echo "Starting frontend server..."
	cd frontend && npm run dev

# 运行单测并生成报告数据
test-report:
	@echo "Running coverage flow test..."
	go test -v uint_cover_test.go

build:
	go build -o cover-server ./cmd/cover-server/main.go

gen_output:
	mkdir -p output/{bin,conf}