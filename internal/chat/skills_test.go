package chat

import "testing"

func TestSetSkillsCopiesAllow(t *testing.T) {
	c := New("http://example.invalid/v1", "k", "m")
	allow := []string{"read", "glob"}
	c.SetSkills("技能说明", allow)
	allow[0] = "write"
	if len(c.skillTools) != 2 || c.skillTools[0] != "read" {
		t.Fatalf("allow should be copied: %v", c.skillTools)
	}

	c.SetSkills("", nil)
	if c.skillTools != nil {
		t.Fatal("nil allow means unrestricted")
	}
	c.SetSkills("", []string{})
	if c.skillTools == nil {
		t.Fatal("empty non-nil allow must stay restricted")
	}
}

func TestOutboundIncludesSkillNote(t *testing.T) {
	c := New("http://example.invalid/v1", "k", "m")
	c.SetSkills("只读技能", nil)
	msgs := c.outbound(nil)
	if len(msgs) != 1 || msgs[0].Role != "system" || msgs[0].Content != "只读技能" {
		t.Fatalf("got %+v", msgs)
	}
}

func TestIntersectTools(t *testing.T) {
	readonly := []string{"read", "glob", "ls"}
	if got := intersectTools(nil, readonly); len(got) != 3 {
		t.Fatalf("nil allow should keep readonly: %v", got)
	}
	got := intersectTools([]string{"write", "read", "get_current_time"}, readonly)
	if len(got) != 1 || got[0] != "read" {
		t.Fatalf("intersection %v", got)
	}
	empty := intersectTools([]string{"write"}, readonly)
	if empty == nil || len(empty) != 0 {
		t.Fatalf("no common tools should be empty non-nil: %#v", empty)
	}
}
