@echo off
REM 双击或在 cmd 中运行：把当前目录（或指定）的 .env 写入全局配置与用户环境变量
set ENVFILE=.env
if not "%~1"=="" set ENVFILE=%~1
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0setup-env.ps1" -EnvFile "%ENVFILE%"
exit /b %ERRORLEVEL%
