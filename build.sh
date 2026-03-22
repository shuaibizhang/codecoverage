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
    go install github.com/shuaibizhang/goc@v1.0.2

    # 将GOBIN添加到PATH中
    local go_bin="$(go env GOPATH)/bin"
    export PATH=$PATH:$go_bin

    # 验证安装
    log "SUCCESS" "goc安装成功，版本: $(goc version)"
}


# goc构建时注入四元组信息到template
export MODULE="${MODULE:-}"
export BRANCH="${BRANCH:-}"
export COMMIT="${COMMIT:-}"
export BASE_COMMIT="${BASE_COMMIT:-}"
set_module_info() {
    log "HEADER" "环境检测"
    log "INFO" "正在检测环境和分支信息..."
    
    # 如果已经通过环境变量传入了元数据（例如在 Docker 构建中），则直接使用
    if [ -n "$MODULE" ] && [ -n "$BRANCH" ] && [ -n "$COMMIT" ]; then
        log "INFO" "检测到已传入的环境变量，跳过自动探测。"
        log "INFO" "仓库模块: $MODULE"
        log "INFO" "当前分支: $BRANCH"
        log "INFO" "当前提交: $COMMIT"
        log "INFO" "基准提交: ${BASE_COMMIT:-'未找到'}"
        return
    fi

    local target_base_branch=${1:-main}

    if [ -n "$GITHUB_ACTIONS" ]; then
        log "INFO" "正在 GitHub Actions 环境中运行"
        MODULE="$GITHUB_REPOSITORY"
        COMMIT="$GITHUB_SHA"

        if [ "$GITHUB_EVENT_NAME" = "pull_request" ]; then
            log "INFO" "检测到 Pull Request 事件"
            BRANCH="$GITHUB_HEAD_REF"
            local base_ref="$GITHUB_BASE_REF"
            
            # 1. 优先尝试从 GitHub Event Payload 获取基准提交 (base sha)
            # 这样即使是浅克隆 (shallow clone) 也能准确找到 PR 的起始点
            if [ -f "$GITHUB_EVENT_PATH" ] && command -v jq >/dev/null 2>&1; then
                BASE_COMMIT=$(jq -r .pull_request.base.sha "$GITHUB_EVENT_PATH" 2>/dev/null || echo "")
                if [ -n "$BASE_COMMIT" ] && [ "$BASE_COMMIT" != "null" ]; then
                    log "INFO" "从 Event Payload 获取到基准提交: $BASE_COMMIT"
                fi
            fi

            # 2. 如果 Payload 获取失败，再尝试 git fetch 和 merge-base
            if [ -z "$BASE_COMMIT" ]; then
                log "INFO" "正在获取 origin/$base_ref..."
                git fetch origin "$base_ref" --depth=1 || true
                BASE_COMMIT=$(git merge-base "origin/$base_ref" "$COMMIT" 2>/dev/null || echo "")
            fi
        else
            log "INFO" "检测到 Push/其他 事件"
            BRANCH="$GITHUB_REF_NAME"
            
            # 尝试自动探测默认分支
            local base_branch=$target_base_branch
            
            # 如果是默认传参 main，且在 GitHub Actions 中，尝试从事件 payload 获取真正的默认分支
            if [ "$base_branch" = "main" ] && [ -f "$GITHUB_EVENT_PATH" ]; then
                log "INFO" "正在尝试从 GitHub Event Payload 探测默认分支..."
                local payload_default
                if command -v jq >/dev/null 2>&1; then
                    payload_default=$(jq -r .repository.default_branch "$GITHUB_EVENT_PATH" 2>/dev/null || echo "")
                fi
                
                if [ -n "$payload_default" ] && [ "$payload_default" != "null" ]; then
                    base_branch="$payload_default"
                    log "INFO" "自动探测到默认分支: $base_branch"
                fi
            fi

            log "INFO" "正在尝试获取基准分支 origin/$base_branch..."
            if ! git fetch origin "$base_branch" --depth=1 2>/dev/null; then
                if [ "$base_branch" = "main" ]; then
                    log "WARN" "未找到 main 分支，尝试获取 master..."
                    base_branch="master"
                    if ! git fetch origin "$base_branch" --depth=1 2>/dev/null; then
                        log "ERROR" "无法获取 main 或 master 分支，无法计算基准提交。"
                    fi
                elif [ "$base_branch" = "master" ]; then
                    log "WARN" "未找到 master 分支，尝试获取 main..."
                    base_branch="main"
                    if ! git fetch origin "$base_branch" --depth=1 2>/dev/null; then
                        log "ERROR" "无法获取 master 或 main 分支，无法计算基准提交。"
                    fi
                else
                    log "ERROR" "无法获取指定的基准分支 origin/$base_branch"
                fi
            fi
            
            # 只有在 fetch 成功的情况下才尝试计算 merge-base
            if git show-ref --verify --quiet "refs/remotes/origin/$base_branch"; then
                BASE_COMMIT=$(git merge-base "origin/$base_branch" "$COMMIT" 2>/dev/null || echo "")
            fi
        fi
    else
        log "INFO" "正在本地环境中运行"
        MODULE=$(basename "$(git rev-parse --show-toplevel 2>/dev/null || pwd)")
        BRANCH=$(git rev-parse --abbrev-ref HEAD)
        COMMIT=$(git rev-parse HEAD)
        
        # 本地计算 merge-base
        local base_branch=$target_base_branch
        if ! git rev-parse --verify "$base_branch" >/dev/null 2>&1; then
            if [ "$base_branch" = "main" ] && git rev-parse --verify "master" >/dev/null 2>&1; then
                base_branch="master"
            fi
        fi
        
        if git rev-parse --verify "$base_branch" >/dev/null 2>&1; then
            BASE_COMMIT=$(git merge-base "$base_branch" "$COMMIT" 2>/dev/null || echo "")
        fi
    fi

    log "INFO" "仓库模块: $MODULE"
    log "INFO" "当前分支: $BRANCH"
    log "INFO" "当前提交: $COMMIT"
    log "INFO" "基准提交: ${BASE_COMMIT:-'未找到'}"
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
            # 注入服务四元组元信息（模块，分支，commit，basecommit）
            set_module_info
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
