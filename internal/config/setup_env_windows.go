//go:build windows

package config

import (
	"fmt"
	"os"
	"sort"

	"golang.org/x/sys/windows/registry"
)

// setUserEnv 写入 Windows 用户级环境变量（对新开终端生效）。
func setUserEnv(kv map[string]string) (keys []string, hint string, err error) {
	keys = sortedKeys(kv)
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE)
	if err != nil {
		return keys, "", fmt.Errorf("打开用户 Environment 注册表失败：%w", err)
	}
	defer key.Close()

	for _, k := range keys {
		v := kv[k]
		if err := key.SetStringValue(k, v); err != nil {
			return keys, "", fmt.Errorf("写入 %s 失败：%w", k, err)
		}
		_ = os.Setenv(k, v)
	}
	hint = "已写入 Windows 用户环境变量。请关闭并重新打开终端（或重新登录）后再运行 hi-agent。"
	return keys, hint, nil
}

func sortedKeys(kv map[string]string) []string {
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
