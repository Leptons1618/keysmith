package core

import (
	"runtime"
	"testing"
)

// Windows-only coverage: CheckAgent manages the ssh-agent service directly
// via PowerShell, and AddToAgent goes through runTool("ssh-add", ...) like
// everywhere else.
func TestCheckAgentWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}

	restore := fakeTool(t, func(name string, args []string) (string, int) {
		return "Running", 0 // canned powershell output
	})
	defer restore()
	if r := CheckAgent(); !r.OK {
		t.Errorf("service Running should mean available: %+v", r)
	}

	restore = fakeTool(t, func(name string, args []string) (string, int) {
		return "cannot be opened", 1
	})
	defer restore()
	r := CheckAgent()
	if r.OK {
		t.Errorf("failed start should not be OK: %+v", r)
	}
	if !contains(r.Message, "Could not start SSH agent") {
		t.Errorf("unexpected message: %q", r.Message)
	}
}

func TestAddToAgentWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}
	withTempHOME(t)

	requireSSHKeygen(t)
	if r := GenerateKey(AlgoEd25519, "id_agent_test_win", "", "", false); !r.OK {
		t.Fatalf("generate failed: %+v", r)
	}

	restore := fakeTool(t, func(name string, args []string) (string, int) { return "", 0 })
	defer restore()
	if r := AddToAgent("id_agent_test_win"); !r.OK || !contains(r.Message, "Added") {
		t.Errorf("success path: %+v", r)
	}

	restore = fakeTool(t, func(name string, args []string) (string, int) {
		return "Could not open a connection to your authentication agent.", 2
	})
	defer restore()
	r := AddToAgent("id_agent_test_win")
	if r.OK || !contains(r.Message, "agent is not running") {
		t.Errorf("no-agent path: %+v", r)
	}
}
