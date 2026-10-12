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

func TestKindGUIFactoryRegistered(t *testing.T) {
	// 不 Create 真实 Fyne 窗口（无头/CI 可能无显示）；只确认工厂已注册。
	mu.RLock()
	fn, ok := factories[KindGUI]
	mu.RUnlock()
	if !ok || fn == nil {
		t.Fatal("KindGUI factory not registered")
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
