# 打包双击即开的桌面 GUI 版：dist\Hi-agent.exe（无控制台黑窗）。
# 用法（仓库根目录）：
#   .\scripts\build-gui.ps1                      # 只生成 exe
#   .\scripts\build-gui.ps1 -Desktop             # 复制到桌面并建快捷方式
#   .\scripts\build-gui.ps1 -Desktop -WorkDir D:\code\myproj

param(
    [switch]$Desktop,
    [string]$WorkDir = $env:USERPROFILE
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$dist = Join-Path $root "dist"
New-Item -ItemType Directory -Force -Path $dist | Out-Null
$exe = Join-Path $dist "Hi-agent.exe"

Write-Host "Building $exe ..."
go build -ldflags "-s -w -H=windowsgui -X main.defaultUI=gui" -o $exe ./cmd/hi-agent
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host "OK: $exe"

# 顺带装一份到 GOPATH\bin，名字不同，避免和终端版 hi-agent.exe 搞混
$gopathBin = Join-Path (go env GOPATH) "bin"
New-Item -ItemType Directory -Force -Path $gopathBin | Out-Null
$guiBin = Join-Path $gopathBin "Hi-agent-gui.exe"
Copy-Item -Force $exe $guiBin
Write-Host "OK: $guiBin  (注意：终端里敲 hi-agent 仍是 TUI；桌面版请双击本文件或 dist\Hi-agent.exe)"

if ($Desktop) {
    $desk = [Environment]::GetFolderPath("Desktop")
    $deskExe = Join-Path $desk "Hi-agent.exe"
    Copy-Item -Force $exe $deskExe
    $lnk = Join-Path $desk "Hi-agent.lnk"
    $shell = New-Object -ComObject WScript.Shell
    $sc = $shell.CreateShortcut($lnk)
    $sc.TargetPath = $deskExe
    $sc.WorkingDirectory = $WorkDir
    $sc.Description = "Hi-agent desktop GUI"
    $sc.Save()
    Write-Host "OK: $deskExe"
    Write-Host "OK: $lnk (WorkingDirectory=$WorkDir)"
}

Write-Host ""
Write-Host "Double-click: dist\Hi-agent.exe  or  Desktop\Hi-agent.exe"
Write-Host "NOT: typing 'hi-agent' in terminal (that is TUI)."
