// Package all 汇总编译期插件；由 cmd/hi-agent 空白导入以完成注册。
package all

import (
	_ "github.com/xgk1906806hjy/Hi-Agent/plugins/echo"
	_ "github.com/xgk1906806hjy/Hi-Agent/plugins/web"
)
