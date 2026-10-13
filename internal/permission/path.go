package permission

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SafePath 把 p（相对 root 或绝对）转成绝对路径，解析符号链接后确认仍在 root 内。
// 目标不存在时解析最近的已存在祖先目录，再拼回剩余部分（用于新建文件）。
func SafePath(root, p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		p = "."
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rootReal, err := resolve(rootAbs)
	if err != nil {
		return "", err
	}
	abs := p
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(rootAbs, abs)
	}
	abs = filepath.Clean(abs)
	real, err := resolve(abs)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootReal, real)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("路径越界：%s 不在工作区 %s 内", p, rootAbs)
	}
	return abs, nil
}

// resolve 解析符号链接；路径不存在时向上找到已存在的祖先解析后再拼接剩余段。
func resolve(abs string) (string, error) {
	rest := ""
	cur := abs
	for {
		real, err := filepath.EvalSymlinks(cur)
		if err == nil {
			if rest == "" {
				return real, nil
			}
			return filepath.Join(real, rest), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return abs, nil
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
}
