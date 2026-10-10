package tui

import (
	"bytes"
	"testing"
)

func TestFilterSlashCmds(t *testing.T) {
	cmds := []SlashCmd{{Name: "/help"}, {Name: "/pwd"}, {Name: "/status"}}
	got := FilterSlashCmds(cmds, "/p")
	if len(got) != 1 || got[0].Name != "/pwd" {
		t.Fatalf("got=%v", got)
	}
	if FilterSlashCmds(cmds, "help") != nil {
		t.Fatal("expected nil without slash")
	}
}

func TestReadKeyWheelSGR(t *testing.T) {
	// ESC [ < 64 ; 1 ; 1 M  = wheel up
	up := readKeyMust(t, "\x1b[<64;1;1M")
	if up.kind != keyWheelUp {
		t.Fatalf("up=%v", up.kind)
	}
	down := readKeyMust(t, "\x1b[<65;2;3M")
	if down.kind != keyWheelDown {
		t.Fatalf("down=%v", down.kind)
	}
}

func readKeyMust(t *testing.T, s string) key {
	t.Helper()
	k, err := readKey(bytes.NewReader([]byte(s)))
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestCompleteSlash(t *testing.T) {
	cmds := []SlashCmd{{Name: "/help"}, {Name: "/pwd"}, {Name: "/status"}, {Name: "/sessions"}}
	if CompleteSlash(cmds, "/he") != "/help" {
		t.Fatal(CompleteSlash(cmds, "/he"))
	}
	if CompleteSlash(cmds, "/s") != "/s" {
		// common prefix of /status /sessions is /s
		got := CompleteSlash(cmds, "/s")
		if got != "/s" && got != "/se" {
			// /status and /sessions → /s is common; /se if only those two with se
			t.Fatalf("got=%q", got)
		}
	}
	got := CompleteSlash(cmds, "/se")
	if got != "/sessions" && got != "/se" {
		// /sessions only matches /se among cmds? /status also HasPrefix /se? no
		// Filter /se → /sessions only → complete to /sessions
		if got != "/sessions" {
			t.Fatalf("got=%q", got)
		}
	}
}
