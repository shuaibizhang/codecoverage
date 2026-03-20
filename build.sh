#!/bin/bash

set -ex

APP=cover-server
OUTPUT=output
CONF_PATH=conf

# 构建
function build() {
    make build
}

# 打包到output目录中
function make_output() {
    if [ -d $OUTPUT ]; then 
        rm -rf $OUTPUT
    fi
    make gen_output
    # 打包二进制产物
    cp $APP $OUTPUT/bin/
    # 打包配置文件
    cp -rf $CONF_PATH $OUTPUT
    # 打包启停脚本
    cp control.sh $OUTPUT
}

build && make_output 