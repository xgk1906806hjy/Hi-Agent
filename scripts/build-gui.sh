#!/usr/bin/env bash
# 打包默认进入桌面 GUI 的可执行文件：dist/hi-agent-gui
# 用法（仓库根目录）：./scripts/build-gui.sh
set -euo pipefail
cd "$(dirname "$0")/.."

mkdir -p dist
go build -ldflags "-s -w -X main.defaultUI=gui" -o dist/hi-agent-gui ./cmd/hi-agent
echo "已生成：$(pwd)/dist/hi-agent-gui"
echo "运行：./dist/hi-agent-gui [项目目录]   （不带参数则以当前目录为工作区）"
