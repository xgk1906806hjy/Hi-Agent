// Package web 是编译期插件示例：提供 /web 提示，说明浏览器模式入口。
package web

import (
	"github.com/xgk1906806hjy/Hi-Agent/internal/plugins"
)

func init() {
	plugins.Register(plugins.Plugin{
		Name:        "web",
		Description: "浏览器模式提示（HTTP + SSE）",
		Commands: []plugins.Command{{
			Name:        "web",
			Description: "显示 Web 模式启动方式与地址",
			Run: func(ctx *plugins.Context, args []string) error {
				ctx.Info("Web 模式：GEEKAGENT_UI=web hi-agent\n" +
					"默认地址 http://127.0.0.1:8787，可用 GEEKAGENT_WEB_ADDR 修改。\n" +
					"当前工作目录：" + ctx.WorkDir())
				return nil
			},
		}},
	})
}
