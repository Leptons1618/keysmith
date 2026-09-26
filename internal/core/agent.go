package core

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// CheckAgent probes (and on Windows starts) the SSH agent.
// Exit codes 0 (agent, no keys) and 1 (agent, keys) both mean available.
func CheckAgent() Result {
	return CheckAgentContext(context.Background())
}

// CheckAgentContext probes the agent and can be cancelled by interactive
// frontends while the service or ssh-add probe is running.
func CheckAgentContext(ctx context.Context) Result {
	if runtime.GOOS == "windows" {
		script := "$service = Get-Service -Name ssh-agent -ErrorAction Stop; " +
			"if ($service.Status -ne 'Running') { Start-Service ssh-agent }; " +
			"(Get-Service -Name ssh-agent).Status"
		out, err := runCmd(runToolContext(ctx, "powershell", "-NoProfile", "-Command", script))
		if err == nil && strings.Contains(out, "Running") {
			return Result{OK: true, Message: "SSH agent is running."}
		}
		return Result{Message: "Could not start SSH agent: " + trimSpace(out)}
	}

	_, err := runCmd(runToolContext(ctx, "ssh-add", "-l"))
	// 0 = agent with no keys, 1 = agent with keys; anything else is unreachable.
	if err == nil || (isExitError(err) && exitCodeOf(err) <= 1) {
		return Result{OK: true, Message: "SSH agent is available."}
	}
	return Result{Message: "SSH agent is not reachable in this session. Start it in your terminal and relaunch the app."}
}

// AddToAgent loads a private key into the agent via ssh-add.
func AddToAgent(keyName string) Result {
	return AddToAgentContext(context.Background(), keyName)
}

// AgentKey is one key currently held by the SSH agent.
type AgentKey struct {
	Bits        int
	Fingerprint string
	Comment     string
	Kind        string
}

// AgentState is the full picture of the agent: whether it answers, and which
// keys it holds. It is what both frontends render on their agent panels.
type AgentState struct {
	Reachable bool
	Keys      []AgentKey
	Raw       string
}

// AgentInfo inspects the agent and returns its current contents. It never
// returns an error: an unreachable agent is reported as Reachable=false so
// callers can render that as a state rather than as a failure.
func AgentInfo(ctx context.Context) AgentState {
	out, err := runCmd(runToolContext(ctx, "ssh-add", "-l"))
	out = trimSpace(out)

	// ssh-add -l exits 0 with an empty agent, 1 with keys, 2 when there is no
	// agent to talk to at all.
	if err == nil {
		return AgentState{Reachable: true, Raw: out}
	}
	if isExitError(err) && exitCodeOf(err) == 1 && out != "" {
		return AgentState{Reachable: true, Keys: parseAgentKeys(out), Raw: out}
	}
	return AgentState{Raw: out}
}

// parseAgentKeys reads "256 SHA256:… you@laptop (ED25519)" lines.
func parseAgentKeys(out string) []AgentKey {
	var keys []AgentKey
	for _, line := range strings.Split(out, "\n") {
		line = trimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		k := AgentKey{}
		if bits, err := strconv.Atoi(fields[0]); err == nil {
			k.Bits = bits
		}
		k.Fingerprint = fields[1]
		rest := fields[2:]
		if close := strings.LastIndex(line, "("); close >= 0 && strings.HasSuffix(line, ")") {
			k.Kind = strings.ToUpper(line[close+1 : len(line)-1])
		}
		k.Comment = strings.TrimSpace(strings.Join(rest, " "))
		keys = append(keys, k)
	}
	return keys
}

// AddToAgentContext loads a private key and stops ssh-add when ctx is cancelled.
func AddToAgentContext(ctx context.Context, keyName string) Result {
	if !ValidKeyName(keyName) {
		return Result{Message: ErrInvalidKeyName.Error()}
	}
	keyPath := PrivateKeyPath(keyName)
	if _, err := os.Stat(keyPath); err != nil {
		return Result{Message: "Private key not found: " + keyPath}
	}

	out, err := runCmd(runToolContext(ctx, "ssh-add", keyPath))
	if err == nil {
		return Result{OK: true, Message: "Added '" + keyName + "' to SSH agent."}
	}
	if ctx.Err() != nil {
		return Result{Message: "Adding key to agent cancelled."}
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
