#!/bin/bash

set -e
# set -x

APP_SERVER=cover-server
APP_AGENT=cover-agent
OUTPUT=output
CONF_PATH=conf

# --- 统一日志函数 ---
# Usage: log <LEVEL> <MESSAGE>
# Levels: INFO, SUCCESS, WARN, ERROR, HEADER, EMPTY
log() {
    local datetime=$(date '+%Y-%m-%d %H:%M:%S')
    # --- 颜色定义 ---
    local red='\033[0;31m'
    local green='\033[0;32m'
    local yellow='\033[0;33m'
    local blue='\033[0;34m'
    local purple='\033[0;35m'
    local cyan='\033[0;36m'
    local no_color='\033[0m' # No Color

    # 日志级别定义
    local level=$1
    local msg=$2
    case "$level" in
        "INFO")    printf "[${datetime}] ${blue}[INFO]${no_color} %s\n" "$msg" ;;
        "SUCCESS") printf "[${datetime}] ${green}[SUCCESS]${no_color} %s\n" "$msg" ;;
        "WARN")    printf "[${datetime}] ${yellow}[WARN]${no_color} %s\n" "$msg" ;;
        "ERROR")   printf "[${datetime}] ${red}[ERROR]${no_color} %s\n" "$msg" >&2 ;;
        "HEADER")  printf "${purple}========== %s ==========${no_color}\n" "$msg" ;;
        "EMPTY")   printf "\n" ;;
        *)         printf "%s\n" "$msg" ;;
    esac
}

# 安装goc 
function install_goc() {
    # 安装goc
    log "INFO" "安装goc中..."
    go install github.com/shuaibizhang/goc@v1.0.1

    # 将GOBIN添加到PATH中
    local go_bin="$(go env GOPATH)/bin"
    export PATH=$PATH:$go_bin

    # 验证安装
    log "SUCCESS" "goc安装成功，版本: $(goc version)"
}

# 准备输出目录
function prepare_output() {
    log "INFO" "准备输出目录: $OUTPUT"
    if [ ! -d $OUTPUT ]; then
        mkdir -p $OUTPUT/bin
        mkdir -p $OUTPUT/conf
    fi
    log "SUCCESS" "输出目录准备完成: $OUTPUT"
}

# 构建 cover-server
function build_server() {
    prepare_output
    log "INFO" "开始构建 cover-server...，插桩编译参数COV=$COV"
    make $APP_SERVER
    log "SUCCESS" "cover-server构建完成: $OUTPUT/bin/$APP_SERVER"
    # 根据 Makefile，二进制可能在根目录或 cmd 目录下
    if [ -f $APP_SERVER ]; then
        cp $APP_SERVER $OUTPUT/bin/
    elif [ -f cmd/$APP_SERVER/$APP_SERVER ]; then
        cp cmd/$APP_SERVER/$APP_SERVER $OUTPUT/bin/
    fi
}

# 构建 cover-agent
function build_agent() {
    prepare_output
    log "INFO" "开始构建 cover-agent..."
    make $APP_AGENT
    log "SUCCESS" "cover-agent构建完成: $OUTPUT/bin/$APP_AGENT"
    # 根据 Makefile，二进制可能在根目录或 cmd 目录下
    if [ -f $APP_AGENT ]; then
        cp $APP_AGENT $OUTPUT/bin/
    elif [ -f cmd/$APP_AGENT/$APP_AGENT ]; then
        cp cmd/$APP_AGENT/$APP_AGENT $OUTPUT/bin/
    fi
}

# 打包配置文件和脚本
function make_output() {
    # 打包配置文件
    if [ -d $CONF_PATH ]; then
        cp -rf $CONF_PATH $OUTPUT
    else
        mkdir -p $OUTPUT/conf
    fi
    # 打包启停脚本
    if [ -f control.sh ]; then
        cp control.sh $OUTPUT
    fi
}

# 清理
function clean() {
    if [ -d $OUTPUT ]; then 
        rm -rf $OUTPUT
    fi
}

# 主逻辑
case "$1" in
    server)
        # 安装goc
        if [ x$COV == xyes ]; then
            install_goc
        fi
        build_server
        make_output
        ;;
    agent)
        build_agent
        make_output
        ;;
    all|*)
        clean
        build_server
        build_agent
        make_output
        ;;
esac