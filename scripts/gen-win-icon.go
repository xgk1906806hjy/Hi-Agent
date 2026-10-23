//go:build ignore

// 从 internal/gui/build/appicon.ico 生成 cmd/hi-agent/rsrc_windows_amd64.syso（资源 ID 3，供 Wails 任务栏图标）。
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tc-hib/winres"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	icoPath := filepath.Join(root, "internal", "gui", "build", "appicon.ico")
	outPath := filepath.Join(root, "cmd", "hi-agent", "rsrc_windows_amd64.syso")
	f, err := os.Open(icoPath)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	ico, err := winres.LoadICO(f)
	if err != nil {
		fatal(err)
	}
	rs := winres.ResourceSet{}
	if err := rs.SetIcon(winres.ID(3), ico); err != nil {
		fatal(err)
	}
	out, err := os.Create(outPath)
	if err != nil {
		fatal(err)
	}
	defer out.Close()
	if err := rs.WriteObject(out, winres.ArchAMD64); err != nil {
		fatal(err)
	}
	fmt.Println("wrote", outPath)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
