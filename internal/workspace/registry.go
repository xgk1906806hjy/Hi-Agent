// Package workspace 维护用户登记过的项目目录，并只读列出各项目的会话。
//
// 不依赖任何界面包；REPL 用它组装 ui.Nav，桌面界面只消费结果。
package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type registryFile struct {
	Projects []string `json:"projects"`
}

// RegistryPath 返回 ~/.hi-agent/projects.json；取不到主目录时为空。
func RegistryPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".hi-agent", "projects.json")
}

// Load 读取已登记项目（最近使用在前）；文件缺失或损坏时返回空列表。
func Load() []string {
	return loadFrom(RegistryPath())
}

// Remember 把目录登记到列表最前面（已存在则前移）。
func Remember(dir string) error {
	return rememberIn(RegistryPath(), dir)
}

// Remove 从列表移除目录；不存在时无操作。
func Remove(dir string) error {
	return removeIn(RegistryPath(), dir)
}

func loadFrom(path string) []string {
	if path == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var f registryFile
	if json.Unmarshal(b, &f) != nil {
		return nil
	}
	return dedupe(f.Projects)
}

func rememberIn(path, dir string) error {
	dir = clean(dir)
	if dir == "" {
		return nil
	}
	next := []string{dir}
	for _, p := range loadFrom(path) {
		if !SamePath(p, dir) {
			next = append(next, p)
		}
	}
	return saveTo(path, next)
}

func removeIn(path, dir string) error {
	dir = clean(dir)
	list := loadFrom(path)
	next := list[:0]
	for _, p := range list {
		if !SamePath(p, dir) {
			next = append(next, p)
		}
	}
	return saveTo(path, next)
}

func saveTo(path string, list []string) error {
	if path == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(registryFile{Projects: list}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func dedupe(in []string) []string {
	out := make([]string, 0, len(in))
	for _, p := range in {
		p = clean(p)
		if p == "" {
			continue
		}
		dup := false
		for _, e := range out {
			if SamePath(e, p) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, p)
		}
	}
	return out
}

func clean(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = filepath.Clean(p)
	if p == "." {
		return ""
	}
	return p
}

// SamePath 比较两个目录是否相同（Windows 下忽略大小写）。
func SamePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// Name 返回侧栏显示用的目录名。
func Name(path string) string {
	base := filepath.Base(path)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return path
	}
	return base
}
