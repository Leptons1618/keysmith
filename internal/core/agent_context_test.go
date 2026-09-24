package core

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestAddToAgentContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("blocking helper uses a POSIX process")
	}
	withTempHOME(t)
	if err := os.MkdirAll(SSHDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(PrivateKeyPath("cancel_me"), []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}

	orig := runToolContext
	runToolContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestBlockingHelperProcess", "--")
		cmd.Env = append(os.Environ(), "KEYSMITH_BLOCKING_HELPER=1")
		return cmd
	}
	defer func() { runToolContext = orig }()

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	result := AddToAgentContext(ctx, "cancel_me")
	if result.OK || !strings.Contains(strings.ToLower(result.Message), "cancel") {
		t.Fatalf("expected cancellation result, got %+v", result)
	}
}
