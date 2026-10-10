package tui

import "testing"

func TestThinkStreamBasic(t *testing.T) {
	var s ThinkStream
	think, model := s.Feed("hello <think>secret</think> world")
	if think != "secret" {
		t.Fatalf("think=%q", think)
	}
	if model != "hello  world" {
		t.Fatalf("model=%q", model)
	}
}

func TestThinkStreamChunked(t *testing.T) {
	var s ThinkStream
	t1, m1 := s.Feed("a <th")
	if t1 != "" || m1 != "a " {
		t.Fatalf("partial open: think=%q model=%q", t1, m1)
	}
	t2, m2 := s.Feed("ink>abc</thi")
	if t2 != "abc" || m2 != "" {
		t.Fatalf("mid: think=%q model=%q", t2, m2)
	}
	t3, m3 := s.Feed("nk>done")
	if t3 != "" || m3 != "done" {
		t.Fatalf("close: think=%q model=%q", t3, m3)
	}
}

func TestClassifyAndSummary(t *testing.T) {
	k, name := ClassifyProgress("[调用工具 ls → files]")
	if k != ProgressTool || name != "ls" {
		t.Fatalf("got %v %q", k, name)
	}
	sum := FoldSummary(2, 1, 0, 0, "ls", true)
	if sum != "▸ 工具×2 · 思考（已折叠）" {
		t.Fatalf("sum=%q", sum)
	}
}

func TestShortProgressLine(t *testing.T) {
	got := ShortProgressLine("[调用工具 read → huge...]")
	if got != "▸ 工具 read" {
		t.Fatalf("got=%q", got)
	}
}
