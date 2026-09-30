package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvChainFromRootEndsWithWorkDir(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(wd)
	if err != nil {
		t.Fatal(err)
	}
	chain := envChainFromRoot(abs)
	if len(chain) == 0 {
		t.Fatal("empty chain")
	}
	if chain[len(chain)-1] != abs {
		t.Fatalf("last=%q want %q", chain[len(chain)-1], abs)
	}
	if filepath.Dir(chain[0]) != chain[0] && len(chain) > 1 {
		// 第一项应是卷根或接近根；至少比第二项更短/更高
		if len(chain[0]) > len(chain[1]) {
			t.Fatalf("root-first order broken: %q then %q", chain[0], chain[1])
		}
	}
}

func TestSnapshotRestorePreservesProcessEnv(t *testing.T) {
	const key = "OPENAI_API_KEY"
	old, had := os.LookupEnv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})

	_ = os.Setenv(key, "from-process")
	snaps := snapshotEnv([]string{key})
	_ = os.Setenv(key, "from-file")
	restoreEnv(snaps)
	if got := os.Getenv(key); got != "from-process" {
		t.Fatalf("got %q", got)
	}
}

func TestRestoreKeepsDotenvWhenUnsetOriginally(t *testing.T) {
	const key = "OPENAI_MODEL"
	old, had := os.LookupEnv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})

	_ = os.Unsetenv(key)
	snaps := snapshotEnv([]string{key})
	_ = os.Setenv(key, "from-dotenv")
	restoreEnv(snaps)
	if got := os.Getenv(key); got != "from-dotenv" {
		t.Fatalf("dotenv value wiped: %q", got)
	}
}
