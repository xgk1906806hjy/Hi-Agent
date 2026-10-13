#!/usr/bin/env bash
# 打包默认进入桌面 GUI 的可执行文件（Wails + React）：dist/hi-agent-gui
# 前置：Node 18+；Linux 需 webkit2gtk 开发包。
# 用法（仓库根目录）：./scripts/build-gui.sh
set -euo pipefail
cd "$(dirname "$0")/.."

fe=internal/gui/frontend
echo "Building frontend ..."
(
  cd "$fe"
  if [[ -f package-lock.json ]]; then npm ci; else npm install; fi
  npm run build
)

mkdir -p dist
go build -tags "desktop,production" -ldflags "-s -w -X main.defaultUI=gui" -o dist/hi-agent-gui ./cmd/hi-agent
echo "已生成：$(pwd)/dist/hi-agent-gui"
echo "运行：./dist/hi-agent-gui [项目目录]   （不带参数则以当前目录为工作区）"
