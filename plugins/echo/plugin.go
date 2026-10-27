// Package echo 是编译期插件示例：注册 /echo 命令并演示生命周期钩子。
package echo

import (
	"strings"

	"github.com/xgk1906806hjy/Hi-Agent/internal/plugins"
)

func init() {
	plugins.Register(plugins.Plugin{
		Name:        "echo",
		Description: "回显参数（编译期插件示例）",
		Commands: []plugins.Command{{
			Name:        "echo",
			Description: "回显参数：/echo <文本>",
			Run: func(ctx *plugins.Context, args []string) error {
				if len(args) == 0 {
					ctx.Info("用法：/echo <文本>")
					return nil
				}
				ctx.Info(strings.Join(args, " "))
				return nil
			},
		}},
	})
}
