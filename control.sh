set -ex

APP_SERVER=bin/cover-server
APP_AGENT=bin/cover-agent
CONF=""

choose_conf_file() {
    local cluster="dev"
    if [[ "x$cluster" == "xdev" ]]; then 
        if [[ "x$SERVICE_TYPE" == "xagent" ]]; then
            CONF="conf/agent-dev.toml"
        else
            CONF="conf/server-dev.toml"
        fi
    fi
    echo "use conf: $CONF"
}

# 启动脚本
function start_server(){
    exec $APP_SERVER -conf $CONF 
}

function start_agent(){
    # 假设agent启动参数和server类似，或者根据需要调整
    exec $APP_AGENT -c $CONF 
}

function stop_server(){
    pkill -f "$APP_SERVER -conf" || true
    echo "Server stopped."
}

function stop_agent(){
    pkill -f "$APP_AGENT -c" || true
    echo "Agent stopped."
}

main() {
    action=$1 
    case $action in 
        "start" )
            choose_conf_file
            if [[ "x$SERVICE_TYPE" == "xagent" ]]; then
                start_agent
            else
                start_server
            fi
        ;;
        "stop" )
            if [[ "x$SERVICE_TYPE" == "xagent" ]]; then
                stop_agent
            else
                stop_server
            fi
        ;;
        "restart" )
            if [[ "x$SERVICE_TYPE" == "xagent" ]]; then
                stop_agent
                sleep 1
                choose_conf_file
                start_agent
            else
                stop_server
                sleep 1
                choose_conf_file
                start_server
            fi
        ;;
        * )
            echo "unknown command"
            exit 1 
        ;;
    esac
}

main $1