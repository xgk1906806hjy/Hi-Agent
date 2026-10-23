package tools

import "testing"

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
