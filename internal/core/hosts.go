package core

import (
	"context"
	"fmt"
	"strings"
)

// TestService runs SSH authentication against one service, port 22 first and
// the vendor 443 endpoint when port 22 looks network-blocked.
func TestService(svc Service, keyName string) HostResult {
	return TestServiceContext(context.Background(), svc, keyName)
}

// TestServiceContext is the cancellable connection-test operation used by
// interactive frontends.
func TestServiceContext(ctx context.Context, svc Service, keyName string) HostResult {
	var keyPath string
	if keyName != "" {
		if _, err := stat(PrivateKeyPath(keyName)); err == nil {
			keyPath = PrivateKeyPath(keyName)
		}
	}
	return testHostContext(ctx, svc, keyPath)
}

func testHost(svc Service, keyPath string) HostResult {
	return testHostContext(context.Background(), svc, keyPath)
}

// testHost tries port 22 first and falls back to the vendor 443 endpoint
// when the failure looks like a blocked network path.
func testHostContext(ctx context.Context, svc Service, keyPath string) HostResult {
	base := []string{"-T", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new", "-o", "ConnectTimeout=10"}
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
		out, err := runCmd(runToolContext(ctx, "ssh", args...))
		out = trimSpace(out)
		if out == "" {
			if ctx.Err() != nil {
				out = ctx.Err().Error()
			} else {
				out = "No output"
			}
		}
		if err != nil && ctx.Err() != nil {
			out = ctx.Err().Error()
		}
		return attempt{label: label, text: out}
	}

	attempts := []attempt{run(nil, "port 22")}
	first := attempts[0]
	if ctx.Err() != nil {
		return HostResult{Host: svc.Host, Output: "Connection test cancelled: " + ctx.Err().Error()}
	}
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
	if strings.Contains(low, "welcome to gitlab") {
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
