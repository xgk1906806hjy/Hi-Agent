#!/usr/bin/env bash
# 将 hi-agent 安装到 $(go env GOPATH)/bin，并尽量把该目录写入 shell 启动脚本。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

echo "Building and installing hi-agent ..."
go install ./cmd/hi-agent
bin="$(go env GOPATH)/bin"
echo "Installed: ${bin}/hi-agent"

marker_begin="# >>> hi-agent path >>>"
marker_end="# <<< hi-agent path <<<"
snippet="${marker_begin}
case \":\$PATH:\" in *\":${bin}:\"*) ;; *) export PATH=\"\$PATH:${bin}\" ;; esac
${marker_end}"

ensure_path_block() {
  local file="$1"
  if [[ -f "$file" ]] && grep -qF "$marker_begin" "$file" 2>/dev/null; then
    return 0
  fi
  if [[ -f "$file" && -s "$file" ]]; then
    local last
    last="$(tail -c1 "$file" || true)"
    if [[ "$last" != $'\n' && -n "$last" ]]; then
      echo >>"$file"
    fi
  fi
  printf '%s\n' "$snippet" >>"$file"
  echo "已写入 PATH 到 $file"
}

case ":$PATH:" in
  *":${bin}:"*)
    echo "当前 PATH 已包含 ${bin}。"
    ;;
  *)
    export PATH="$PATH:${bin}"
    echo "已在当前会话加入 PATH。"
    ;;
esac

for rc in "${HOME}/.profile" "${HOME}/.bashrc" "${HOME}/.zshrc"; do
  ensure_path_block "$rc" || true
done

global_dir="${HOME}/.hi-agent"
global_env="${global_dir}/.env"
if [[ ! -f "$global_env" ]]; then
  mkdir -p "$global_dir"
  cat >"$global_env" <<'EOF'
OPENAI_BASE_URL=https://api.deepseek.com
OPENAI_API_KEY=
OPENAI_MODEL=deepseek-v4-flash
EOF
  echo "已创建全局配置模板：${global_env} （请填写 OPENAI_API_KEY，或运行 ./scripts/setup-env.sh）"
else
  echo "全局配置：${global_env}"
fi

echo
echo "请重开终端（或 source ~/.bashrc），然后：cd 任意项目目录后执行 hi-agent"
