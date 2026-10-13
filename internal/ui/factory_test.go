package ui

import "testing"

func TestNormalizeAndResolve(t *testing.T) {
	if normalizeKind("TEXT") != KindPlain {
		t.Fatal(normalizeKind("TEXT"))
	}
	if normalizeKind("fullscreen") != KindTUI {
		t.Fatal(normalizeKind("fullscreen"))
	}
}

func TestRegisterAndCreatePlain(t *testing.T) {
	u, err := Create(KindPlain, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if u == nil {
		t.Fatal("nil ui")
	}
	u.Close()
}

func TestKindGUIPlaceholderWithoutDesktop(t *testing.T) {
	// 未带 desktop 标签时 gui 只有占位工厂，应给出明确错误而非未注册。
	if _, err := Create(KindGUI, Options{}); err == nil {
		t.Fatal("expected placeholder error")
	}
}

func TestListKindsContainsBuiltins(t *testing.T) {
	kinds := ListKinds()
	seen := map[Kind]bool{}
	for _, k := range kinds {
		seen[k] = true
	}
	for _, need := range []Kind{KindPlain, KindTUI, KindGUI, KindWeb} {
		if !seen[need] {
			t.Fatalf("missing %s in %v", need, kinds)
		}
	}
}
