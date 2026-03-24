# 此dockerfile为多阶段构建 （Multi-stage Build）：
# - 第一阶段： FROM golang... AS builder （用于编译代码）
# - 第二阶段： FROM alpine:latest （用于运行程序）
# ARG 的作用域仅限于它被定义的那个阶段。你在第一阶段定义的 SERVICE_TYPE 在进入第二阶段（ FROM alpine ）后就 失效 了。因此，必须在第二阶段重新声明一次 ARG SERVICE_TYPE ，才能拿到构建时传入的值。

# 构建阶段
FROM golang:1.25-alpine AS builder

# 构建参数，默认为 server
ARG SERVICE_TYPE=server
# 如果需要开启覆盖率构建，可以传入 COV=yes
ARG COV=no

# 注入构建元数据
ARG MODULE=""
ARG BRANCH=""
ARG COMMIT=""
ARG BASE_COMMIT=""
ARG GITHUB_ACTIONS=""

# 设置工作目录
WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache make bash git

# 复制项目代码
COPY . .

# 执行构建脚本，根据 SERVICE_TYPE 构建对应的服务
RUN MODULE=${MODULE} \
    BRANCH=${BRANCH} \
    COMMIT=${COMMIT} \
    BASE_COMMIT=${BASE_COMMIT} \
    GITHUB_ACTIONS=${GITHUB_ACTIONS} \
    COV=${COV} ./build.sh ${SERVICE_TYPE}

# 运行阶段
FROM alpine:latest

# 运行时变量
ARG SERVICE_TYPE=server
ENV SERVICE_TYPE=${SERVICE_TYPE}

# 设置工作目录
WORKDIR /app

# 创建日志目录
RUN mkdir -p /app/logs && chmod 777 /app/logs

# 复制构建产物
COPY --from=builder /app/output /app/

# 安装必要的运行时依赖
RUN apk add --no-cache bash libc6-compat

# 暴露所有服务端口
# cover-server: 8080 (HTTP), 9090 (gRPC)
# cover-agent: 8180 (HTTP), 2039 (RPC/Config)
EXPOSE 8080 9090 8180 2039

# 默认启动命令，可以通过环境变量 SERVICE_TYPE 控制启动的服务
ENTRYPOINT ["/bin/bash", "./control.sh", "start"]
