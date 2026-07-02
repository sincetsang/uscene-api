#!/bin/bash

C=$1

if [ -z $C ]; then
  echo "no target"
  exit 2
fi

if [ ! -d cmd/$C ]; then
  echo "build cmd/$C: not found."
  exit 2
fi

# 确保输出目录存在
mkdir -p out

### 编译并启动
go build -o "out/$C" "./cmd/$C" && "out/$C" "${@:2}"