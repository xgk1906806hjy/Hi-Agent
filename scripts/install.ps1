# 将 hi-agent 安装到 GOPATH/bin，并把该目录写入用户 PATH（永久）。
# 用法：在仓库根目录执行  .\scripts\install.ps1

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

Write-Host "Building and installing hi-agent ..."
go install ./cmd/hi-agent
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$bin = Join-Path (go env GOPATH) "bin"
$exe = Join-Path $bin "hi-agent.exe"
Write-Host "Installed: $exe"

# 永久加入用户 PATH（若尚未包含）
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not $userPath) { $userPath = "" }
$dirs = $userPath -split ";" | Where-Object { $_ -ne "" }
$already = $false
foreach ($d in $dirs) {
    if ($d.TrimEnd("\") -ieq $bin.TrimEnd("\")) { $already = $true; break }
}
if (-not $already) {
    $newPath = if ($userPath.Trim() -eq "") { $bin } else { "$userPath;$bin" }
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Host "已将 $bin 写入用户 PATH（永久）。"
} else {
    Write-Host "用户 PATH 已包含 $bin。"
}

# 当前会话立刻可用
if (($env:Path -split ";") -notcontains $bin) {
    $env:Path = "$env:Path;$bin"
}

$globalEnvDir = Join-Path $env:USERPROFILE ".hi-agent"
$globalEnv = Join-Path $globalEnvDir ".env"
if (-not (Test-Path $globalEnv)) {
    New-Item -ItemType Directory -Force -Path $globalEnvDir | Out-Null
    @"
OPENAI_BASE_URL=https://api.deepseek.com
OPENAI_API_KEY=
OPENAI_MODEL=deepseek-v4-flash
"@ | Set-Content -Path $globalEnv -Encoding utf8
    Write-Host "已创建全局配置模板：$globalEnv （请填写 OPENAI_API_KEY，或运行 .\scripts\setup-env.ps1）"
} else {
    Write-Host "全局配置：$globalEnv"
}

Write-Host ""
Write-Host "请关闭并重新打开终端，然后：cd 任意项目目录后执行  hi-agent"
