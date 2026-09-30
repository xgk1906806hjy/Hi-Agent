# 从 .env 一键配置：写入 ~/.hi-agent/.env，并设置 Windows 用户环境变量。
# 用法：
#   .\scripts\setup-env.ps1
#   .\scripts\setup-env.ps1 -EnvFile D:\path\to\.env

param(
    [string]$EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $EnvFile)) {
    Write-Host "找不到 $EnvFile"
    Write-Host "请先复制 .env.example 为 .env 并填写 OPENAI_API_KEY，再运行本脚本。"
    exit 1
}

$AbsEnv = (Resolve-Path -LiteralPath $EnvFile).Path

if (Get-Command hi-agent -ErrorAction SilentlyContinue) {
    & hi-agent setup-env $AbsEnv
    exit $LASTEXITCODE
}

$root = Split-Path -Parent $PSScriptRoot
if (Test-Path (Join-Path $root "cmd\hi-agent")) {
    Push-Location $root
    try {
        go run ./cmd/hi-agent setup-env $AbsEnv
        exit $LASTEXITCODE
    } finally {
        Pop-Location
    }
}

Write-Host "未找到 hi-agent，且不在仓库内。请先：go install ./cmd/hi-agent"
exit 1
