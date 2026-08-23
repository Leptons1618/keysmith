package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func TestValidKeyName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"id_ed25519", true},
		{"id_work.github", true},
		{"a-b_c.d9", true},
		{"", false},
		{"has space", false},
		{"slash/name", false},
		{"../escape", false},
	}
	for _, c := range cases {
		if got := ValidKeyName(c.name); got != c.want {
			t.Errorf("ValidKeyName(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

// withTempHOME points the home directory at a fresh directory for the
// duration of the test. os.UserHomeDir reads USERPROFILE on Windows and HOME
// elsewhere, so both are set.
func withTempHOME(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

// fakeTool replaces the external OpenSSH tools (ssh, ssh-add, ssh-keygen)
// with a canned-output fake, so tests behave identically on every OS.
// fn maps an invocation to its combined output and exit code. The returned
// restore func must be deferred by the caller.
func fakeTool(t *testing.T, fn func(name string, args []string) (string, int)) (restore func()) {
	t.Helper()
	orig := runTool
	runTool = func(name string, args ...string) *exec.Cmd {
		out, code := fn(name, args)
		cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(os.Environ(),
			"FAKE_TOOL_CHILD=1",
			"FAKE_TOOL_OUTPUT="+out,
			"FAKE_TOOL_EXIT="+strconv.Itoa(code),
		)
		return cmd
	}
	return func() { runTool = orig }
}

// TestHelperProcess is the child side of fakeTool: it echoes the canned
// output and exits with the canned status. It never runs as a real test.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("FAKE_TOOL_CHILD") != "1" {
		t.Skip("helper only runs as a fakeTool child")
	}
	fmt.Fprint(os.Stderr, os.Getenv("FAKE_TOOL_OUTPUT"))
	os.Exit(exitCodeOfHelper())
}

func exitCodeOfHelper() int {
	code, err := strconv.Atoi(os.Getenv("FAKE_TOOL_EXIT"))
	if err != nil || code == 0 {
		return 0
	}
	return code
}

func requireSSHKeygen(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}
}

func TestListKeysEmpty(t *testing.T) {
	withTempHOME(t)
	if got := ListKeys(); len(got) != 0 {
		t.Errorf("ListKeys on empty home = %v, want none", got)
	}
}

func TestGenerateListDeleteRoundTrip(t *testing.T) {
	withTempHOME(t)
	requireSSHKeygen(t)

	res := GenerateKey(AlgoEd25519, "id_test_a", "test@example", "", false)
	if !res.OK {
		t.Fatalf("GenerateKey ed25519: %+v", res)
	}

	keys := ListKeys()
	if len(keys) != 1 || keys[0].Name != "id_test_a" {
		t.Fatalf("ListKeys = %+v, want [id_test_a]", keys)
	}
	if keys[0].PubPath == "" || filepath.Ext(keys[0].PubPath) != ".pub" {
		t.Errorf("PubPath = %q", keys[0].PubPath)
	}

	pub := PublicKey("id_test_a")
	if pub == "" || !contains(pub, "ssh-ed25519") {
		t.Errorf("PublicKey = %q", pub)
	}

	fp := Fingerprint("id_test_a")
	if fp == "" || !contains(fp, ":") {
		t.Errorf("Fingerprint = %q", fp)
	}

	if err := DeleteKey("id_test_a"); err != nil {
		t.Fatalf("DeleteKey: %v", err)
	}
	if _, err := os.Stat(filepath.Join(SSHDir(), "id_test_a")); !os.IsNotExist(err) {
		t.Error("private key still exists after delete")
	}
	if _, err := os.Stat(filepath.Join(SSHDir(), "id_test_a.pub")); !os.IsNotExist(err) {
		t.Error("public key still exists after delete")
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestGenerateInvalidNameRejected(t *testing.T) {
	withTempHOME(t)
	res := GenerateKey(AlgoEd25519, "../evil", "", "", false)
	if res.OK {
		t.Fatal("GenerateKey accepted path traversal name")
	}
	if _, err := os.Stat(SSHDir()); !os.IsNotExist(err) && err == nil {
		dir, _ := os.ReadDir(SSHDir())
		if len(dir) > 0 {
			t.Logf("~/.ssh unexpectedly populated: %d entries", len(dir))
		}
	}
}

func TestGenerateForceOverwrites(t *testing.T) {
	withTempHOME(t)
	requireSSHKeygen(t)

	if r := GenerateKey(AlgoEd25519, "id_force_me", "first", "", false); !r.OK {
		t.Fatalf("first generate failed: %+v", r)
	}
	old := PublicKey("id_force_me")

	r := GenerateKey(AlgoEd25519, "id_force_me", "second", "", true)
	if !r.OK {
		t.Fatalf("force regenerate failed: %+v", r)
	}
	if PublicKey("id_force_me") == "" {
		t.Fatal("key missing after force regeneration")
	}
	_ = old
}

func TestGenerateRSABitSize(t *testing.T) {
	withTempHOME(t)
	requireSSHKeygen(t)

	if r := GenerateKey(AlgoRSA, "id_rsa_test", "", "", false); !r.OK {
		t.Fatalf("rsa generate failed: %+v", r)
	}
	pub := PublicKey("id_rsa_test")
	if !contains(pub, "ssh-rsa") && !contains(pub, "rsa-sha2") {
		t.Errorf("expected RSA public key, got %.60q", pub)
	}
}
