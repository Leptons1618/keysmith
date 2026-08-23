package core

import (
	"fmt"
	"os/exec"
	"strings"
)

// TestService runs SSH authentication against one service, port 22 first and
// the vendor 443 endpoint when port 22 looks network-blocked.
func TestService(svc Service, keyName string) HostResult {
	var keyPath string
	if keyName != "" {
		if _, err := stat(PrivateKeyPath(keyName)); err == nil {
			keyPath = PrivateKeyPath(keyName)
		}
	}
	return testHost(svc, keyPath)
}

// testHost tries port 22 first and falls back to the vendor 443 endpoint
// when the failure looks like a blocked network path.
func testHost(svc Service, keyPath string) HostResult {
	base := []string{"-T", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new"}
	if keyPath != "" {
		base = append(base, "-i", keyPath, "-o", "IdentitiesOnly=yes")
	}

	type attempt struct {
		label string
		text  string
	}
	run := func(extra []string, label string) attempt {
		args := append(append([]string{}, base...), extra...)
		args = append(args, "git@"+svc.Host)
		out, _ := runCmd(exec.Command("ssh", args...))
		out = trimSpace(out)
		if out == "" {
			out = "No output"
		}
		return attempt{label: label, text: out}
	}

	attempts := []attempt{run(nil, "port 22")}
	first := attempts[0]
	if ok := isSuccessOutput(first.text); ok {
		return HostResult{Host: svc.Host, OK: true, Output: fmt.Sprintf("OK (%s)\n%s", first.label, first.text)}
	}

	blocked := isNetworkBlockedOutput(first.text)
	if blocked && svc.AltHost != "" && svc.AltPort != "" {
		extra := []string{"-p", svc.AltPort, "-o", "Hostname=" + svc.AltHost}
		label := fmt.Sprintf("port %s (%s)", svc.AltPort, svc.AltHost)
		attempts = append(attempts, run(extra, label))
		for _, a := range attempts[1:] {
			if ok := isSuccessOutput(a.text); ok {
				return HostResult{Host: svc.Host, OK: true, Output: fmt.Sprintf("OK (%s)\n%s", a.label, a.text)}
			}
		}
	}

	var parts []string
	for _, a := range attempts {
		parts = append(parts, fmt.Sprintf("--- %s ---\n%s", a.label, a.text))
	}
	return HostResult{Host: svc.Host, Output: strings.Join(parts, "\n\n")}
}

// isSuccessOutput reports whether ssh output indicates successful authentication.
func isSuccessOutput(text string) bool {
	low := strings.ToLower(text)
	if strings.Contains(low, "successfully authenticated") {
		return true
	}
	return strings.Contains(low, "authenticated") && strings.Contains(low, "shell access")
}

// isNetworkBlockedOutput reports whether ssh failed for network reasons,
// which is what triggers the port-443 fallback endpoints.
func isNetworkBlockedOutput(text string) bool {
	low := strings.ToLower(text)
	needles := []string{
		"connection timed out",
		"operation timed out",
		"no route to host",
		"connection refused",
		"could not resolve hostname",
		"network is unreachable",
		"connect to host",
	}
	for _, n := range needles {
		if strings.Contains(low, n) {
			return true
		}
	}
	return false
}
