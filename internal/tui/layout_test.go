package tui

import "testing"

// TestTextWidthCJK 校验 CJK 按 2 列计入显示宽度。
func TestTextWidthCJK(t *testing.T) {
	if got := textWidth("ab中文"); got != 6 {
		t.Fatalf("width=%d want 6", got)
	}
}

// TestWrapRespectsWidth 校验折行后每行显示宽度不超过上限。
func TestWrapRespectsWidth(t *testing.T) {
	lines := wrap("你好世界hello", 6)
	for _, l := range lines {
		if textWidth(l) > 6 {
			t.Fatalf("line %q exceeds width", l)
		}
	}
	if len(lines) != 3 {
		t.Fatalf("lines=%v", lines)
	}
}

// TestWrapKeepsNewlines 校验空行（连续换行）被保留。
func TestWrapKeepsNewlines(t *testing.T) {
	lines := wrap("a\n\nb", 10)
	if len(lines) != 3 || lines[1] != "" {
		t.Fatalf("lines=%q", lines)
	}
}

// TestFitPadsAndTruncates 校验 fit 按显示宽度截断并右侧补空格。
func TestFitPadsAndTruncates(t *testing.T) {
	if got := fit("中文字", 5); textWidth(got) != 5 {
		t.Fatalf("fit width=%d (%q)", textWidth(got), got)
	}
	if got := fit("ab", 4); got != "ab  " {
		t.Fatalf("fit=%q", got)
	}
}

// TestSanitizeStripsEscape 校验 ESC/\r 丢弃、\t 展开为 4 空格。
func TestSanitizeStripsEscape(t *testing.T) {
	if got := sanitize("a\x1b[31mb\tc\r"); got != "a[31mb    c" {
		t.Fatalf("sanitize=%q", got)
	}
}

// TestVisibleLinesTakesTail 校验可视区只保留最新若干行。
func TestVisibleLinesTakesTail(t *testing.T) {
	tu := &TUI{}
	for i := 0; i < 5; i++ {
		tu.entries = append(tu.entries, entry{style: "sys", text: string(rune('a' + i))})
	}
	got := tu.visibleLines(10, 2)
	if len(got) != 2 || got[0].text != "d" || got[1].text != "e" {
		t.Fatalf("got=%+v", got)
	}
}
