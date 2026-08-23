package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// SSHDir returns the user's ~/.ssh directory.
func SSHDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ssh"
	}
	return filepath.Join(home, ".ssh")
}

var keyNameRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ValidKeyName reports whether name may be used as an ssh-keygen file name.
func ValidKeyName(name string) bool {
	return keyNameRe.MatchString(name)
}

// ListKeys scans ~/.ssh/*.pub and keeps pairs whose private file exists,
// sorted case-insensitively by name, mirroring list_ssh_keys().
func ListKeys() []KeyInfo {
	dir := SSHDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var keys []KeyInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".pub" {
			continue
		}
		base := name[:len(name)-len(".pub")]
		priv := filepath.Join(dir, base)
		if st, err := os.Stat(priv); err == nil && !st.IsDir() {
			keys = append(keys, KeyInfo{
				Name:    base,
				Path:    priv,
				PubPath: filepath.Join(dir, name),
			})
		}
	}
	sortKeys(keys)
	return keys
}

func sortKeys(keys []KeyInfo) {
	// insertion sort keeps this dependency-free; n is tiny
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && lower(keys[j].Name) < lower(keys[j-1].Name); j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if 'A' <= b[i] && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

// GenerateKey runs ssh-keygen. force deletes an existing pair first.
// Note: like the original implementation, the passphrase is passed via argv
// (-N) and is briefly visible in the process list; parity was chosen over a
// rework so both versions behave identically during migration.
func GenerateKey(algo KeyAlgorithm, keyName, comment, passphrase string, force bool) Result {
	switch algo {
	case AlgoEd25519, AlgoRSA, AlgoECDSA:
	default:
		return Result{Message: fmt.Sprintf("Unsupported algorithm: %s", algo)}
	}
	if !ValidKeyName(keyName) {
		return Result{Message: "Key generation failed: invalid key name"}
	}

	dir := SSHDir()
	_ = os.MkdirAll(dir, 0o700)
	keyPath := filepath.Join(dir, keyName)
	pubPath := keyPath + ".pub"

	if force {
		_ = os.Remove(keyPath)
		_ = os.Remove(pubPath)
	}

	args := []string{"-t", string(algo), "-f", keyPath, "-N", passphrase}
	if algo == AlgoRSA {
		args = append(args, "-b", "4096")
	}
	if comment != "" {
		args = append(args, "-C", comment)
	}

	out, err := runCmd(exec.Command("ssh-keygen", args...))
	if err != nil {
		return Result{Message: fmt.Sprintf("Key generation failed: %s", out)}
	}
	return Result{OK: true, Message: fmt.Sprintf("Key '%s' generated successfully.", keyName)}
}

// PublicKey loads the public key text for keyName, or "" when missing.
func PublicKey(keyName string) string {
	if keyName == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(SSHDir(), keyName+".pub"))
	if err != nil {
		return ""
	}
	return trimSpace(string(data))
}

// Fingerprint returns the ssh-keygen -lf fingerprint line, or "".
func Fingerprint(keyName string) string {
	pubPath := filepath.Join(SSHDir(), keyName+".pub")
	if _, err := os.Stat(pubPath); err != nil {
		return ""
	}
	out, err := runCmd(exec.Command("ssh-keygen", "-lf", pubPath))
	if err != nil {
		return ""
	}
	return trimSpace(out)
}

// PrivateKeyPath is the private key path for keyName under ~/.ssh.
func PrivateKeyPath(keyName string) string {
	return filepath.Join(SSHDir(), keyName)
}

// DeleteKey removes both key files from ~/.ssh.
func DeleteKey(keyName string) error {
	_ = os.Remove(PrivateKeyPath(keyName))
	return os.Remove(PrivateKeyPath(keyName) + ".pub")
}

// runCmd captures combined output and normalizes CRLF on Windows.
func runCmd(cmd *exec.Cmd) (string, error) {
	out, err := cmd.CombinedOutput()
	if runtime.GOOS == "windows" {
		out = []byte(normalizeNewlines(string(out)))
	}
	return string(out), err
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}
