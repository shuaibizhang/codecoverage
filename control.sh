set -ex

APP=bin/cover-server
CONF=""

choose_conf_file() {
    local cluster="dev"
    if [[ "x$cluster" == "xdev" ]]; then 
        CONF="conf/dev.toml"
    fi
    echo "use conf: $CONF"
}

# 启动脚本
function start(){
    # 使用指定命令替换当前的shell进程，以非子进程方式启动，这样可以方便信号传递
    exec $APP -conf $CONF
}

main() {
    action=$1 
    case $action in 
        "start" )
            choose_conf_file
            start 
        ;;
        
        * )
            echo "unknown command"
            exit 1 
        ;;
    esac
}

main $1