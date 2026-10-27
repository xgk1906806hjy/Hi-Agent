package tools

import "testing"

func TestUnregister(t *testing.T) {
	const name = "test_unregister_dynamic"
	Register(Tool{Name: name, Description: "test", Parameters: map[string]any{}, Run: func(string) (string, error) { return "ok", nil }})
	t.Cleanup(func() { Unregister(name) })
	found := false
	for _, n := range Names() {
		if n == name {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("注册后未找到 %s", name)
	}
	Unregister(name)
	for _, n := range Names() {
		if n == name {
			t.Fatalf("Unregister 后仍存在 %s", name)
		}
	}
	Unregister(name) // 不存在时不应 panic
}

func TestToOpenAIAllowNilVsEmpty(t *testing.T) {
	if got := ToOpenAIAllow(nil); len(got) != len(Names()) {
		t.Fatalf("nil allow should expose all: %d", len(got))
	}
	if got := ToOpenAIAllow([]string{}); len(got) != 0 {
		t.Fatalf("empty non-nil allow should expose none: %d", len(got))
	}
	if got := ToOpenAIAllow([]string{"read"}); len(got) != 1 || got[0].Function.Name != "read" {
		t.Fatalf("filtered allow: %+v", got)
	}
}
