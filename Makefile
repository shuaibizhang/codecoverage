GO = go
GOC = goc

# 链接时添加构建信息
VERSION = 1.0.0
BUILD_TIME = $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
BUILDINFO_PKG = github.com/shuaibizhang/codecoverage/internal/buildinfo
LDFLAGS = -ldflags '-X $(BUILDINFO_PKG).Version=$(VERSION) -X $(BUILDINFO_PKG).BuildTime=$(BUILD_TIME)'
# 添加gc参数					
GCFLAGS = -x 
ifeq ($(COV),yes)
	GCFLAGS += -gcflags='all=-N -l'
endif

.PHONY: all cover-server cover-agent cover-cli run-backend run-frontend test-report minio-up minio-down help

# ================== 构建命令 ===================
all: cover-server cover-agent cover-cli

cover-server:
ifeq ($(COV),yes)
	cd ./cmd/$@ && $(GOC) build --center=http://127.0.0.1:2039 --agentport=:7778 --buildflags="$(GCFLAGS) $(LDFLAGS)" -o $@ 
else
	$(GO) build $(GCFLAGS) -o $@ ./cmd/$@
endif

cover-agent:
	$(GO) build $(GCFLAGS) -o $@ ./cmd/$@

cover-cli:
	$(GO) build $(GCFLAGS) -o $@ ./cmd/$@

gen_output:
	mkdir -p output/{bin,conf}

# ================= 其他测试命令 ====================
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
	@echo "Starting backend server via go run cmd/cover-server/main.go ..."
	go run cmd/cover-server/main.go 2>&1 | tee -a backend.log

# 启动前端服务器
run-frontend:
	@echo "Starting frontend server..."
	cd frontend && npm run dev

# 运行单测并生成报告数据
test-report:
	@echo "Running coverage flow test..."
	go test -v uint_cover_test.go

# 端口转发，18080 端口映射到本地 8080 端口 （服务未部署时使用）
# 服务部署后，无需使用
remote_port_forward:
	@echo "Stopping existing SSH tunnels..."
	-pkill -f "ssh -fCNR 0.0.0.0:18080"
	@echo "Starting enhanced SSH tunnel with keepalive..."
	ssh -fCNR 0.0.0.0:18080:localhost:8080 \
		-o ServerAliveInterval=30 \
		-o ServerAliveCountMax=3 \
		-o ExitOnForwardFailure=yes \
		bingoserver
	@echo "Starting backend server..."
	make run-backend

test_remote_port_forward:
	@echo "Testing remote port forward..."
	curl -v http://49.233.216.158:18080/api/v1/coverage/file
