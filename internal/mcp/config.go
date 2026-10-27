package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigPath 是项目级 MCP 配置文件，相对工作区根目录。
const ConfigPath = ".geekagent/mcp.json"

// serverConfig 对应 mcp.json 中的一个 server。
type serverConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
	Cwd     string            `json:"cwd"`
}

type configFile struct {
	// 兼容两种常见写法：mcpServers（优先）与 servers。
	MCPServers map[string]serverConfig `json:"mcpServers"`
	Servers    map[string]serverConfig `json:"servers"`
}

func (c configFile) entries() map[string]serverConfig {
	if len(c.MCPServers) > 0 {
		return c.MCPServers
	}
	return c.Servers
}

// loadConfig 读取工作区下的 MCP 配置；文件不存在返回 found=false 且 err=nil。
func loadConfig(root string) (cfg map[string]serverConfig, found bool, err error) {
	path := filepath.Join(root, filepath.FromSlash(ConfigPath))
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, fmt.Errorf("读取 %s 失败：%w", ConfigPath, err)
	}
	var file configFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, true, fmt.Errorf("解析 %s 失败：%w", ConfigPath, err)
	}
	return file.entries(), true, nil
}
