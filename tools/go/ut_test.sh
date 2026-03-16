#!/bin/bash

# -e 确保脚本在出错时退出
# -x 方便调试
set -e
# set -x

# --- 统一日志函数 ---
# Usage: log <LEVEL> <MESSAGE>
# Levels: INFO, SUCCESS, WARN, ERROR, HEADER, EMPTY
log() {
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
        "INFO")    printf "${blue}[INFO]${no_color} %s\n" "$msg" ;;
        "SUCCESS") printf "${green}[SUCCESS]${no_color} %s\n" "$msg" ;;
        "WARN")    printf "${yellow}[WARN]${no_color} %s\n" "$msg" ;;
        "ERROR")   printf "${red}[ERROR]${no_color} %s\n" "$msg" >&2 ;;
        "HEADER")  printf "${purple}========== %s ==========${no_color}\n" "$msg" ;;
        "EMPTY")   printf "\n" ;;
        *)         printf "%s\n" "$msg" ;;
    esac
}

# flag准备
# go编译参数，禁用内联优化
BUILD_FLAG=(-gcflags 'all=-N -l')
# 测试超时时长5分钟，
TEST_FLAG=(-timeout 5m -vet off)
prepare_flags(){
    log "INFO" "初始化编译参数和测试参数.."

    # 编译参数设置
    # 通过TAG环境变量，设置go build tag，以实现条件编译
    if [ -n "$TAG" ]; then 
        BUILD_FLAG+=(-tags $(TAG))
    fi
    # 检测下BUILD_FLAG_RACE环境变量，确定是否开启竞态检测
    if [ "$BUILD_FLAG_RACE" = "on" ]; then 
        BUILD_FLAG+=(-race)
    fi
    # 设置并行度
    if [ -n "$BUILD_FLAG_P" ]; then 
        BUILD_FLAG+=(-p ${BUILD_FLAG_P})
    fi

    # go test参数设置
    if [ -n "$TEST_FLAG_PARALLEL" ]; then 
        TEST_FLAG+=(-parallel ${TEST_FLAG_PARALLEL})
    fi
    log "SUCCESS" "初始化编译参数和测试参数成功"
    log "INFO" "编译参数=${BUILD_FLAG[*]} 测试参数=${TEST_FLAG[*]}"
}

# 设置不执行单测的排除文件夹
PACKAGE_LIST="./..."
get_go_test_package_list(){
    # 每次进入模块时重置为默认值
    PACKAGE_LIST="./..."
    
    if [ -n "$EXCLUDE_DIRS" ]; then 
        log "INFO" "检测到排除目录: ${EXCLUDE_DIRS}，正在生成包列表..."
        # 从EXCLUDE_DIRS环境变量中读取，按照','分割成数组
        IFS=',' read -ra EXCLUDE_DIRS_ARRAY <<< "$EXCLUDE_DIRS"
        # 初始化
        local exclude="go list ./... | grep -v /vendor/ "
        # 遍历数组
        for dir in "${EXCLUDE_DIRS_ARRAY[@]}"; do 
            # 不断拼接 grep -v 目录名
            exclude="$exclude | grep -v $dir"
        done
        PACKAGE_LIST=$(eval "$exclude" 2>/dev/null || echo "")
        
        if [ -n "$PACKAGE_LIST" ]; then
            # 将多行输出转换为单行空格分隔
            PACKAGE_LIST=$(echo "$PACKAGE_LIST" | tr '\n' ' ')
            log "INFO" "待测试包列表: ${PACKAGE_LIST}"
        else
            log "WARN" "排除目录后没有剩余的包可测试。"
            PACKAGE_LIST=""
        fi
    fi
}

export MODULE=""
export BRANCH=""
export COMMIT=""
export BASE_COMMIT=""
export RUN_ID="" # 单元测试执行id
# --- 环境检测函数 ---
set_module_info() {
    log "HEADER" "环境检测"
    log "INFO" "正在检测环境和分支信息..."
    
    local target_base_branch=${1:-main}

    if [ -n "$GITHUB_ACTIONS" ]; then
        log "INFO" "正在 GitHub Actions 环境中运行"
        MODULE="$GITHUB_REPOSITORY"
        COMMIT="$GITHUB_SHA"

        if [ "$GITHUB_EVENT_NAME" = "pull_request" ]; then
            log "INFO" "检测到 Pull Request 事件"
            BRANCH="$GITHUB_HEAD_REF"
            local base_ref="$GITHUB_BASE_REF"
            
            # Fetch 基础分支以计算 merge-base
            log "INFO" "正在获取 origin/$base_ref..."
            git fetch origin "$base_ref" --depth=1 || true
            
            BASE_COMMIT=$(git merge-base "origin/$base_ref" "$COMMIT" 2>/dev/null || echo "")
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

    # 设置一个uuid作为执行ID
    if command -v uuidgen >/dev/null 2>&1; then
        RUN_ID=$(uuidgen | tr '[:upper:]' '[:lower:]')
    else
        RUN_ID=$(cat /proc/sys/kernel/random/uuid 2>/dev/null || python3 -c 'import uuid; print(uuid.uuid4())' 2>/dev/null || echo "manual-$(date +%s)")
    fi

    log "INFO" "执行 ID: $RUN_ID"
    log "INFO" "仓库模块: $MODULE"
    log "INFO" "当前分支: $BRANCH"
    log "INFO" "当前提交: $COMMIT"
    log "INFO" "基准提交: ${BASE_COMMIT:-'未找到'}"
}

# --- 运行测试与生成覆盖率 ---
run_ut_test_with_coverage() {
    log "HEADER" "Go 单元测试"
    
    # 1. 准备全局编译和测试参数
    prepare_flags
    
    log "INFO" "正在查找所有 Go 模块..."
    local module_dirs
    module_dirs=$(find . -name "go.mod" -not -path "*/.*" | xargs dirname | sort -u)

    for mod_dir in $module_dirs; do
        log "INFO" "正在处理模块: $mod_dir"
        # 使用子 shell 切换目录，避免影响后续操作
        (
            cd "$mod_dir"
            
            # 2. 获取当前模块待测试的包列表（考虑排除目录）
            # 注意：get_go_test_package_list 会根据 EXCLUDE_DIRS 环境变量动态生成 PACKAGE_LIST
            get_go_test_package_list
            
            if [ -z "$PACKAGE_LIST" ]; then
                log "WARN" "$mod_dir 模块没有任何待测试的包。"
                return
            fi

            # 3. 执行测试并生成覆盖率
            # 使用数组扩展 "${BUILD_FLAG[@]}" "${TEST_FLAG[@]}" 以正确处理带空格的参数
            # 并将 PACKAGE_LIST 放在最后作为测试目标
            local test_output
            set +e
            test_output=$(go test "${BUILD_FLAG[@]}" "${TEST_FLAG[@]}" -coverprofile=coverage.out ${PACKAGE_LIST} 2>&1)
            local test_exit_code=$?
            set -e

            if [ $test_exit_code -eq 0 ]; then
                if [ -f coverage.out ] && [ -s coverage.out ]; then
                    local total_cov
                    total_cov=$(go tool cover -func=coverage.out | tail -n 1 | awk '{print $NF}')
                    log "SUCCESS" "$mod_dir 模块覆盖率: $total_cov"
                else
                    log "WARN" "$mod_dir 模块未发现测试用例，跳过覆盖率计算。"
                fi
            else
                # 检查是否是因为没有测试文件
                if echo "$test_output" | grep -q "no test files"; then
                    log "WARN" "$mod_dir 模块中未发现测试文件。"
                else
                    log "ERROR" "$mod_dir 模块单元测试失败。"
                    # log "ERROR" "测试输出详情:\n$test_output"
                fi
            fi
        )
    done
}

# --- 生成 Git Diff ---
generate_git_diff() {
    log "HEADER" "Git 差异生成"
    
    if [ -z "$BASE_COMMIT" ]; then
        log "ERROR" "无法找到基准提交 (BASE_COMMIT)，跳过差异生成。"
        return 1
    fi

    log "INFO" "正在基于 $BASE_COMMIT 生成 diff_changes.patch..."
    git diff "$BASE_COMMIT" "$COMMIT" > diff_changes.patch
    log "SUCCESS" "Git 差异已保存至 diff_changes.patch"
}

# --- 主函数 ---
main() {
    local target_base_branch=${1:-main}
    
    set_module_info "$target_base_branch"
    log "EMPTY"
    
    run_ut_test_with_coverage
    log "EMPTY"
    
    if generate_git_diff; then
        log "EMPTY"
        log "HEADER" "执行完成"
        log "SUCCESS" "脚本执行成功！"
    else
        log "EMPTY"
        log "HEADER" "错误"
        log "ERROR" "差异生成过程中执行失败。"
        exit 1
    fi
}

# 执行主流程
main "$@"
