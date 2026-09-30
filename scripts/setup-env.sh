#!/usr/bin/env bash
# 从 .env 一键配置：写入 ~/.hi-agent/.env，并挂到 shell 启动脚本。
# 用法：
#   ./scripts/setup-env.sh
#   ./scripts/setup-env.sh /path/to/.env

set -euo pipefail

ENV_FILE="${1:-.env}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "找不到 $ENV_FILE"
  echo "请先复制 .env.example 为 .env 并填写 OPENAI_API_KEY，再运行本脚本。"
  exit 1
fi

ABS_ENV="$(cd "$(dirname "$ENV_FILE")" && pwd)/$(basename "$ENV_FILE")"

if command -v hi-agent >/dev/null 2>&1; then
  exec hi-agent setup-env "$ABS_ENV"
fi

root="$(cd "$(dirname "$0")/.." && pwd)"
if [[ -d "$root/cmd/hi-agent" ]]; then
  cd "$root"
  exec go run ./cmd/hi-agent setup-env "$ABS_ENV"
fi

echo "未找到 hi-agent，且不在仓库内。请先：go install ./cmd/hi-agent"
exit 1
