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

var gh = func() Service { s, _ := ServiceByID("github"); return s }()
var gl = func() Service { s, _ := ServiceByID("gitlab"); return s }()

func TestServiceRegistryComplete(t *testing.T) {
	wantIDs := map[string]bool{"github": false, "gitlab": false, "bitbucket": false}
	for _, s := range Services {
		if _, ok := wantIDs[s.ID]; !ok {
			t.Errorf("unexpected service %q", s.ID)
		}
		wantIDs[s.ID] = true
		if len(s.Steps) < 5 {
			t.Errorf("%s: too few instruction steps (%d)", s.ID, len(s.Steps))
		}
		if s.Host == "" || s.KeysURL == "" || s.AltHost == "" {
			t.Errorf("%s: missing host/url/alt fields: %+v", s.ID, s)
		}
	}
	for id, seen := range wantIDs {
		if !seen {
			t.Errorf("service %q missing from registry", id)
		}
	}
	if _, ok := ServiceByID("nope"); ok {
		t.Error("ServiceByID should reject unknown ids")
	}
}

func TestTestServiceSuccessOnPort22(t *testing.T) {
	withTempHOME(t)
	restore := fakeTool(t, func(name string, args []string) (string, int) {
		return "Hi username! You've successfully authenticated, but GitHub does not provide shell access.", 0
	})
	defer restore()

	r := TestService(gh, "")
	if !r.OK {
		t.Errorf("expected OK, got %+v", r)
	}
	if r.Host != "github.com" {
		t.Errorf("Host = %q", r.Host)
	}
}

func TestTestServiceGitLabSuccessWording(t *testing.T) {
	withTempHOME(t)
	restore := fakeTool(t, func(name string, args []string) (string, int) {
		return "Welcome to GitLab, @user!", 0
	})
	defer restore()

	r := TestService(gl, "")
	if !r.OK {
		t.Errorf("expected GitLab success, got %+v", r)
	}
}

func TestTestServiceContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("blocking helper uses a POSIX process")
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
	r := TestServiceContext(ctx, gh, "")
	if r.OK || !strings.Contains(strings.ToLower(r.Output), "deadline") {
		t.Errorf("expected cancellation output, got %+v", r)
	}
}

func TestBlockingHelperProcess(t *testing.T) {
	if os.Getenv("KEYSMITH_BLOCKING_HELPER") != "1" {
		t.Skip("helper only")
	}
	time.Sleep(time.Second)
}

func TestTestServiceFallbackTo443(t *testing.T) {
	withTempHOME(t)
	restore := fakeTool(t, func(name string, args []string) (string, int) {
		for _, arg := range args {
			if arg == "Hostname=ssh.github.com" || arg == "Hostname=altssh.gitlab.com" {
				return "Hi! You've successfully authenticated.", 0
			}
		}
		return "ssh: connect to host github.com port 22: Connection timed out", 255
	})
	defer restore()

	for _, svcID := range []string{"github", "gitlab"} {
		svc, _ := ServiceByID(svcID)
		r := TestService(svc, "")
		if !r.OK {
			t.Errorf("%s: 443 fallback not triggered: %.200q", svcID, r.Output)
		}
	}
}

func TestTestServiceFailureAggregatesAttempts(t *testing.T) {
	withTempHOME(t)
	restore := fakeTool(t, func(name string, args []string) (string, int) {
		return "Permission denied (publickey).", 255
	})
	defer restore()

	r := TestService(gh, "")
	if r.OK {
		t.Fatal("unexpectedly OK")
	}
	if !contains(r.Output, "--- port 22 ---") {
		t.Errorf("attempt log missing label, got %.120q", r.Output)
	}
}

func TestCheckAgentExitCodeMapping(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("CheckAgent drives the Windows service directly; see agent_windows_test.go")
	}
	withTempHOME(t)

	restore := fakeTool(t, func(name string, args []string) (string, int) { return "", 1 })
	defer restore()
	if r := CheckAgent(); !r.OK {
		t.Errorf("exit 1 should mean agent available: %+v", r)
	}

	restore = fakeTool(t, func(name string, args []string) (string, int) { return "", 0 })
	defer restore()
	if r := CheckAgent(); !r.OK {
		t.Errorf("exit 0 should mean agent available: %+v", r)
	}

	restore = fakeTool(t, func(name string, args []string) (string, int) {
		return "Could not open a connection to your authentication agent.", 2
	})
	defer restore()
	if r := CheckAgent(); r.OK {
		t.Errorf("exit 2 should mean unreachable: %+v", r)
	} else if !contains(r.Message, "not reachable") {
		t.Errorf("unexpected message: %q", r.Message)
	}
}

func TestAddToAgentPaths(t *testing.T) {
	withTempHOME(t)

	r := AddToAgent("no_such_key")
	if r.OK || !contains(r.Message, "not found") {
		t.Errorf("missing key: %+v", r)
	}

	requireSSHKeygen(t)
	if r := GenerateKey(AlgoEd25519, "id_agent_test", "", "", false); !r.OK {
		t.Fatalf("generate failed: %+v", r)
	}

	restore := fakeTool(t, func(name string, args []string) (string, int) { return "", 0 })
	defer restore()
	if r := AddToAgent("id_agent_test"); !r.OK || !contains(r.Message, "Added") {
		t.Errorf("success path: %+v", r)
	}

	restore = fakeTool(t, func(name string, args []string) (string, int) {
		return "Could not open a connection to your authentication agent.", 2
	})
	defer restore()
	r = AddToAgent("id_agent_test")
	if r.OK || !contains(r.Message, "agent is not running") {
		t.Errorf("no-agent path: %+v", r)
	}
}
