package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
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
// Dot path components are rejected because filepath.Join resolves them as
// directory aliases rather than file names.
func ValidKeyName(name string) bool {
	return name != "." && name != ".." && keyNameRe.MatchString(name)
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

// GenerateKey runs ssh-keygen synchronously for non-interactive callers.
func GenerateKey(algo KeyAlgorithm, keyName, comment, passphrase string, force bool) Result {
	return GenerateKeyContext(context.Background(), algo, keyName, comment, passphrase, force)
}

// GenerateKeyContext runs ssh-keygen and stops it when ctx is cancelled.
func GenerateKeyContext(ctx context.Context, algo KeyAlgorithm, keyName, comment, passphrase string, force bool) Result {
	switch algo {
	case AlgoEd25519, AlgoRSA, AlgoECDSA:
	default:
		return Result{Message: fmt.Sprintf("Unsupported algorithm: %s", algo)}
	}
	if !ValidKeyName(keyName) {
		return Result{Message: "Key generation failed: invalid key name"}
	}

	dir := SSHDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Result{Message: "Key generation failed: could not create ~/.ssh: " + trimSpace(err.Error())}
	}
	keyPath := filepath.Join(dir, keyName)
	pubPath := keyPath + ".pub"

	if force {
		if err := removeIfExists(keyPath); err != nil {
			return Result{Message: "Key generation failed: could not replace private key: " + trimSpace(err.Error())}
		}
		if err := removeIfExists(pubPath); err != nil {
			return Result{Message: "Key generation failed: could not replace public key: " + trimSpace(err.Error())}
		}
	}

	args := []string{"-t", string(algo), "-f", keyPath, "-N", passphrase}
	if algo == AlgoRSA {
		args = append(args, "-b", "4096")
	}
	if comment != "" {
		args = append(args, "-C", comment)
	}

	out, err := runCmd(runToolContext(ctx, "ssh-keygen", args...))
	if err != nil {
		if ctx.Err() != nil {
			return Result{Message: "Key generation cancelled."}
		}
		return Result{Message: fmt.Sprintf("Key generation failed: %s", out)}
	}
	return Result{OK: true, Message: fmt.Sprintf("Key '%s' generated successfully.", keyName)}
}

// PublicKey loads the public key text for keyName, or "" when missing or invalid.
func PublicKey(keyName string) string {
	if !ValidKeyName(keyName) {
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
	if !ValidKeyName(keyName) {
		return ""
	}
	pubPath := filepath.Join(SSHDir(), keyName+".pub")
	if _, err := os.Stat(pubPath); err != nil {
		return ""
	}
	out, err := runCmd(runTool("ssh-keygen", "-lf", pubPath))
	if err != nil {
		return ""
	}
	return trimSpace(out)
}

// ShortFingerprint returns just the SHA256 digest from an ssh-keygen -lf line
// ("256 SHA256:abc… comment (ED25519)" becomes "SHA256:abc…"), or "" when the
// line carries no digest.
func ShortFingerprint(fp string) string {
	for _, field := range strings.Fields(fp) {
		if strings.HasPrefix(field, "SHA256:") {
			return field
		}
	}
	return ""
}

// KeyKind reports the algorithm behind a key as a display label such as
// "Ed25519", "RSA 4096" or "ECDSA". ok is false when the key cannot be
// inspected at all. The ssh-keygen fingerprint line is preferred because it
// carries the key size as well as the algorithm; the public key prefix is the
// fallback when ssh-keygen is unavailable.
func KeyKind(keyName string) (string, bool) {
	if !ValidKeyName(keyName) {
		return "", false
	}
	if label, ok := kindFromFingerprint(Fingerprint(keyName)); ok {
		return label, true
	}
	return kindFromPublicKey(PublicKey(keyName))
}

// kindFromFingerprint parses "4096 SHA256:… comment (RSA)" into "RSA 4096".
// The bit count is only shown where it actually varies: Ed25519 and DSA have a
// single fixed size, and repeating it on every row is noise.
func kindFromFingerprint(fp string) (string, bool) {
	if fp == "" {
		return "", false
	}
	open := strings.LastIndex(fp, "(")
	if open < 0 || !strings.HasSuffix(fp, ")") {
		return "", false
	}
	algo := strings.ToUpper(trimSpace(fp[open+1 : len(fp)-1]))
	if algo == "" {
		return "", false
	}
	if algo == "ED25519" || algo == "DSA" || algo == "ECDSA" {
		return algo, true
	}
	bits := strings.Fields(fp)[0]
	if _, err := strconv.Atoi(bits); err == nil {
		return algo + " " + bits, true
	}
	return algo, true
}

// kindFromPublicKey reads the algorithm straight off the public key blob,
// which always starts with the key type followed by the base64 body.
func kindFromPublicKey(pub string) (string, bool) {
	prefix := strings.Fields(pub)
	if len(prefix) == 0 {
		return "", false
	}
	switch prefix[0] {
	case "ssh-ed25519":
		return "ED25519", true
	case "ssh-rsa":
		return "RSA", true
	case "ssh-dss":
		return "DSA", true
	case "ecdsa-sha2-nistp256":
		return "ECDSA nistp256", true
	case "ecdsa-sha2-nistp384":
		return "ECDSA nistp384", true
	case "ecdsa-sha2-nistp521":
		return "ECDSA nistp521", true
	}
	if strings.HasPrefix(prefix[0], "sk-ssh-ed25519") {
		return "ED25519 (sk)", true
	}
	return "", false
}

// PrivateKeyPath is the private key path for keyName under ~/.ssh.
func PrivateKeyPath(keyName string) string {
	if !ValidKeyName(keyName) {
		return ""
	}
	return filepath.Join(SSHDir(), keyName)
}

// DeleteKey removes both key files from ~/.ssh. Missing files are treated as
// already deleted, while permission and other filesystem failures are returned.
func DeleteKey(keyName string) error {
	if !ValidKeyName(keyName) {
		return ErrInvalidKeyName
	}
	if err := removeIfExists(PrivateKeyPath(keyName)); err != nil {
		return err
	}
	return removeIfExists(PrivateKeyPath(keyName) + ".pub")
}

func removeIfExists(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// runTool builds a command for an external OpenSSH tool (ssh, ssh-add,
// ssh-keygen). It exists so tests can substitute fake tools cross-platform:
// PATH-shim scripts are not executable on Windows, so tests swap runTool
// itself instead.
var runTool = exec.Command

// runToolContext is the cancellable command seam used by interactive
// operations. It remains separate so existing synchronous tests can keep their
// small fake-tool seam.
var runToolContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
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
