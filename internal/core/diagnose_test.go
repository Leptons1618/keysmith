package core

import "testing"

func TestDiagnoseNetworkBlocked(t *testing.T) {
	out := "--- port 22 ---\nssh: connect to host github.com port 22: Connection timed out\n\n--- port 443 ---\nssh: connect to host ssh.github.com port 443: Connection timed out"
	d := Diagnose(out)
	if len(d) == 0 {
		t.Fatal("no diagnosis")
	}
	if !contains(d[0].Cause, "cannot reach") {
		t.Errorf("cause = %q", d[0].Cause)
	}
	if !contains(d[0].Fix, "443") {
		t.Errorf("fix should mention 443 attempt: %q", d[0].Fix)
	}
}

func TestDiagnosePermissionDenied(t *testing.T) {
	out := "--- port 22 ---\ngit@github.com: Permission denied (publickey)."
	d := Diagnose(out)
	if len(d) == 0 {
		t.Fatal("no diagnosis")
	}
	if !contains(d[0].Cause, "did not accept this key") {
		t.Errorf("cause = %q", d[0].Cause)
	}
	if !contains(d[0].Fix, "reopen the keys page") {
		t.Errorf("fix should point at instructions: %q", d[0].Fix)
	}
}

func TestDiagnoseIdentityFileProblem(t *testing.T) {
	out := "Permission denied (publickey).\r\nCould not read identity file /home/x/.ssh/key: no such file"
	d := Diagnose(out)
	if !contains(d[0].Cause, "could not read the private key") {
		t.Errorf("cause = %q", d[0].Cause)
	}
}

func TestDiagnoseAgentUnreachable(t *testing.T) {
	out := "error connecting to agent: Could not open a connection to your authentication agent"
	d := Diagnose(out)
	if !contains(d[0].Cause, "agent is not running") {
		t.Errorf("cause = %q", d[0].Cause)
	}
	if !contains(d[0].Fix, "eval $(ssh-agent)") {
		t.Errorf("fix = %q", d[0].Fix)
	}
}

func TestDiagnoseHostKeyVerification(t *testing.T) {
	out := "Host key verification failed."
	d := Diagnose(out)
	if !contains(d[0].Cause, "host key mismatch") {
		t.Errorf("cause = %q", d[0].Cause)
	}
	if !contains(d[0].Fix, "ssh-keygen -R") {
		t.Errorf("fix = %q", d[0].Fix)
	}
}

func TestDiagnoseGenericFallback(t *testing.T) {
	out := "weird unexpected output"
	d := Diagnose(out)
	if len(d) != 1 || !contains(d[0].Cause, "did not succeed") {
		t.Errorf("expected generic fallback, got %+v", d)
	}
}
