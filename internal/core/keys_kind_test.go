package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- key kind ----------------------------------------------------------------

// TestKindFromFingerprint covers the display labels for each algorithm, and
// pins the rule that a fixed-size algorithm does not show its bit count.
func TestKindFromFingerprint(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"ed25519 drops the bit count", "256 SHA256:abc you@laptop (ED25519)", "ED25519"},
		{"rsa keeps the bit count", "4096 SHA256:abc work@box (RSA)", "RSA 4096"},
		{"rsa 2048", "2048 SHA256:abc old@box (RSA)", "RSA 2048"},
		{"ecdsa drops the bit count", "256 SHA256:abc me@laptop (ECDSA)", "ECDSA"},
		{"dsa drops the bit count", "1024 SHA256:abc legacy (DSA)", "DSA"},
		{"no closing paren", "256 SHA256:abc broken", ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := kindFromFingerprint(tt.line)
			if tt.want == "" {
				if ok {
					t.Fatalf("kindFromFingerprint(%q) = %q, want no result", tt.line, got)
				}
				return
			}
			if !ok || got != tt.want {
				t.Fatalf("kindFromFingerprint(%q) = %q/%v, want %q", tt.line, got, ok, tt.want)
			}
		})
	}
}

// TestKindFromPublicKey covers the fallback used when ssh-keygen is unavailable.
func TestKindFromPublicKey(t *testing.T) {
	tests := []struct {
		name string
		pub  string
		want string
	}{
		{"ed25519", "ssh-ed25519 AAAA user@laptop", "ED25519"},
		{"rsa", "ssh-rsa AAAA user@laptop", "RSA"},
		{"dsa", "ssh-dss AAAA user@laptop", "DSA"},
		{"ecdsa 256", "ecdsa-sha2-nistp256 AAAA user@laptop", "ECDSA nistp256"},
		{"ecdsa 384", "ecdsa-sha2-nistp384 AAAA user@laptop", "ECDSA nistp384"},
		{"ecdsa 521", "ecdsa-sha2-nistp521 AAAA user@laptop", "ECDSA nistp521"},
		{"security key", "sk-ssh-ed25519@openssh.com AAAA user@laptop", "ED25519 (sk)"},
		{"empty", "", ""},
		{"unknown", "not-a-key-type AAAA", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := kindFromPublicKey(tt.pub)
			if tt.want == "" {
				if ok {
					t.Fatalf("kindFromPublicKey(%q) = %q, want no result", tt.pub, got)
				}
				return
			}
			if !ok || got != tt.want {
				t.Fatalf("kindFromPublicKey(%q) = %q/%v, want %q", tt.pub, got, ok, tt.want)
			}
		})
	}
}

func TestKeyKindRejectsInvalidNames(t *testing.T) {
	if _, ok := KeyKind("../escape"); ok {
		t.Error("KeyKind accepted a path-traversing name")
	}
	if _, ok := KeyKind(""); ok {
		t.Error("KeyKind accepted an empty name")
	}
}

func TestKeyKindFallsBackWhenKeygenUnavailable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	dir := SSHDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "k.pub"), []byte("ssh-ed25519 AAAA me@laptop\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Make ssh-keygen fail so the public-key fallback is exercised.
	defer fakeTool(t, func(name string, args []string) (string, int) {
		return "ssh-keygen: not available", 1
	})()

	got, ok := KeyKind("k")
	if !ok || got != "ED25519" {
		t.Fatalf("KeyKind = %q/%v, want ED25519 from the public key", got, ok)
	}
}

// --- short fingerprint ---------------------------------------------------------

func TestShortFingerprint(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"normal", "256 SHA256:abc123 you@laptop (ED25519)", "SHA256:abc123"},
		{"with bits", "4096 SHA256:def456 work@box (RSA)", "SHA256:def456"},
		{"no digest", "256 comment (ED25519)", ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShortFingerprint(tt.in); got != tt.want {
				t.Fatalf("ShortFingerprint(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// --- agent inventory --------------------------------------------------------------

func TestParseAgentKeys(t *testing.T) {
	out := strings.Join([]string{
		"256 SHA256:7Yq1n0K8s3BqWzT4mXpLvC9dF2hJ5rEuA0oIzYwXcVb you@laptop (ED25519)",
		"4096 SHA256:1Ab2Cd3Ef4Gh5Ij6Kl7Mn8Op9Qr0St1Uv2Wx3Yz work@box (RSA)",
	}, "\n")

	keys := parseAgentKeys(out)
	if len(keys) != 2 {
		t.Fatalf("parsed %d keys, want 2", len(keys))
	}
	if keys[0].Bits != 256 || keys[0].Kind != "ED25519" || keys[0].Comment != "you@laptop" {
		t.Errorf("first key = %+v", keys[0])
	}
	if keys[0].Fingerprint != "SHA256:7Yq1n0K8s3BqWzT4mXpLvC9dF2hJ5rEuA0oIzYwXcVb" {
		t.Errorf("first fingerprint = %q", keys[0].Fingerprint)
	}
	if keys[1].Bits != 4096 || keys[1].Kind != "RSA" {
		t.Errorf("second key = %+v", keys[1])
	}
}

func TestParseAgentKeysSkipsNoise(t *testing.T) {
	if got := parseAgentKeys(""); len(got) != 0 {
		t.Errorf("empty input parsed %d keys", len(got))
	}
	if got := parseAgentKeys("The agent has no identities."); len(got) != 0 {
		t.Errorf("prose was parsed as %d keys: %+v", len(got), got)
	}
	// A comment containing parentheses must not be mistaken for the kind.
	keys := parseAgentKeys("256 SHA256:abc my (work) box (ED25519)")
	if len(keys) != 1 || keys[0].Kind != "ED25519" {
		t.Errorf("kind = %+v, want the trailing parenthesised token", keys)
	}
}

// TestAgentInfoExitCodes covers the three states ssh-add -l can report: keys
// present (exit 1 with output), an empty agent (exit 0), and no agent at all.
func TestAgentInfoExitCodes(t *testing.T) {
	tests := []struct {
		name          string
		out           string
		exit          int
		wantReachable bool
		wantKeys      int
	}{
		{"keys present", "256 SHA256:abc you@laptop (ED25519)", 1, true, 1},
		{"empty agent", "", 0, true, 0},
		{"no agent", "Could not open a connection to your authentication agent.", 2, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer fakeTool(t, func(name string, args []string) (string, int) {
				return tt.out, tt.exit
			})()

			got := AgentInfo(t.Context())
			if got.Reachable != tt.wantReachable {
				t.Errorf("Reachable = %v, want %v", got.Reachable, tt.wantReachable)
			}
			if len(got.Keys) != tt.wantKeys {
				t.Errorf("parsed %d keys, want %d", len(got.Keys), tt.wantKeys)
			}
		})
	}
}

func TestLooksLikeFingerprint(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{"sha256 digest", "SHA256:7Yq1n0K8s3BqWzT4mXpLvC9dF2hJ5rEuA0oIzYwXcVb", true},
		{"md5 pair", "aa:bb:cc:dd", true},
		{"long hex", "0123456789abcdef0123456789abcdef", true},
		{"short hex", "abcd", false},
		{"a word", "agent", false},
		{"a number", "256", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeFingerprint(tt.token); got != tt.want {
				t.Fatalf("looksLikeFingerprint(%q) = %v, want %v", tt.token, got, tt.want)
			}
		})
	}
}
