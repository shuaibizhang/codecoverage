set -ex 

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
    log "INFO" "install goc..."
    go install github.com/shuaibizhang/goc@bingozhang

    # 将GOBIN添加到PATH中
    local go_bin = $(go env GOPATH)/bin
    export PATH=$PATH:$go_bin

    # 验证安装
    log "SUCCESS" "goc installed successfully,goc version: $(goc version)"
}