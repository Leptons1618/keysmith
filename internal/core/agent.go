package core

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// CheckAgent probes (and on Windows starts) the SSH agent.
// Exit codes 0 (agent, no keys) and 1 (agent, keys) both mean available.
func CheckAgent() Result {
	if runtime.GOOS == "windows" {
		script := "$service = Get-Service -Name ssh-agent -ErrorAction Stop; " +
			"if ($service.Status -ne 'Running') { Start-Service ssh-agent }; " +
			"(Get-Service -Name ssh-agent).Status"
		out, err := runCmd(exec.Command("powershell", "-NoProfile", "-Command", script))
		if err == nil && strings.Contains(out, "Running") {
			return Result{OK: true, Message: "SSH agent is running."}
		}
		return Result{Message: "Could not start SSH agent: " + trimSpace(out)}
	}

	_, err := runCmd(exec.Command("ssh-add", "-l"))
	// 0 = agent with no keys, 1 = agent with keys; anything else is unreachable.
	if err == nil || (isExitError(err) && exitCodeOf(err) <= 1) {
		return Result{OK: true, Message: "SSH agent is available."}
	}
	return Result{Message: "SSH agent is not reachable in this session. Start it in your terminal and relaunch the app."}
}

// AddToAgent loads a private key into the agent via ssh-add.
func AddToAgent(keyName string) Result {
	keyPath := PrivateKeyPath(keyName)
	if _, err := os.Stat(keyPath); err != nil {
		return Result{Message: "Private key not found: " + keyPath}
	}

	out, err := runCmd(exec.Command("ssh-add", keyPath))
	if err == nil {
		return Result{OK: true, Message: "Added '" + keyName + "' to SSH agent."}
	}
	if strings.Contains(out, "Could not open a connection to your authentication agent") {
		return Result{Message: "SSH agent is not running. Start it first, then retry."}
	}
	return Result{Message: "Failed to add key to agent: " + trimSpace(out)}
}

// trimSpace is strings.TrimSpace under a local alias to keep call sites short.
func trimSpace(s string) string { return strings.TrimSpace(s) }

// isExitError reports whether err is an exec.ExitError (the child ran and
// returned a non-zero status).
func isExitError(err error) bool {
	_, ok := err.(*exec.ExitError)
	return ok
}

// exitCodeOf returns the child process exit code for an exec.ExitError.
func exitCodeOf(err error) int {
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	return -1
}
