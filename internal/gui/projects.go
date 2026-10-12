package gui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type projectFile struct {
	Projects []string `json:"projects"`
}

// ProjectsPath 用户级项目列表（~/.hi-agent/projects.json）。
func ProjectsPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".hi-agent", "projects.json")
}

// LoadProjects 读取已添加的项目目录（绝对路径）。文件不存在返回空列表。
func LoadProjects() []string {
	p := ProjectsPath()
	if p == "" {
		return nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var f projectFile
	if json.Unmarshal(b, &f) != nil {
		return nil
	}
	return cleanProjectList(f.Projects)
}

// RememberProject 把目录记入项目列表（去重，新的放前面）。
func RememberProject(dir string) error {
	dir = filepath.Clean(dir)
	if dir == "" || dir == "." {
		return nil
	}
	list := LoadProjects()
	next := make([]string, 0, len(list)+1)
	next = append(next, dir)
	for _, p := range list {
		if samePath(p, dir) {
			continue
		}
		next = append(next, p)
	}
	return saveProjects(next)
}

func saveProjects(list []string) error {
	p := ProjectsPath()
	if p == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(projectFile{Projects: list}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

func cleanProjectList(in []string) []string {
	out := make([]string, 0, len(in))
	for _, p := range in {
		p = filepath.Clean(strings.TrimSpace(p))
		if p == "" || p == "." {
			continue
		}
		dup := false
		for _, e := range out {
			if samePath(e, p) {
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

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// ProjectName 侧栏显示用的目录名。
func ProjectName(path string) string {
	base := filepath.Base(path)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return path
	}
	return base
}
