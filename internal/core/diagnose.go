package core

import "strings"

// Diagnosis is one identified problem with the exact next step to fix it.
type Diagnosis struct {
	Cause string
	Fix   string
}

// Diagnose maps raw ssh output to human-readable causes and concrete fixes,
// most likely first. It always returns at least one entry.
func Diagnose(output string) []Diagnosis {
	low := strings.ToLower(output)
	var out []Diagnosis

	switch {
	case containsAny(low,
		"connection timed out", "operation timed out", "no route to host",
		"connection refused", "could not resolve hostname", "network is unreachable"):
		out = append(out, Diagnosis{
			Cause: "This machine cannot reach " + hostNameFrom(output) + " on the network.",
			Fix: "Check your internet connection. If you are on a corporate or school network, " +
				"its firewall may block port 22; the app already tried the port-443 endpoint. " +
				"Try a different network (e.g. phone hotspot) or ask IT to allow SSH to the host.",
		})

	case strings.Contains(low, "host key verification failed"):
		out = append(out, Diagnosis{
			Cause: "The server's identity could not be verified (host key mismatch).",
			Fix: "The known_hosts entry is stale. Run: ssh-keygen -R <hostname> then retry; " +
				"the app accepts new host keys automatically on first connect.",
		})

	case strings.Contains(low, "permission denied") && containsAny(low, "publickey", "public key"):
		if containsAny(low, "identity file", "no such file", "unreadable") {
			out = append(out, Diagnosis{
				Cause: "SSH could not read the private key file.",
				Fix:   "Check the key file exists in ~/.ssh and that you can open it. If it has a passphrase, add the key to the agent from the key screen first.",
			})
		} else {
			out = append(out, Diagnosis{
				Cause: "The service did not accept this key — the public key probably was not added (or not saved) on the website.",
				Fix: "Go back to the instructions, press the button to reopen the keys page, and check: " +
					"does an entry exist whose key ends with the same characters as your public key? " +
					"If not, copy the key again and paste it once more, then save and test again.",
			})
		}

	case strings.Contains(low, "could not open a connection to your authentication agent"),
		strings.Contains(low, "agent not running"), strings.Contains(low, "error connecting to agent"):
		out = append(out, Diagnosis{
			Cause: "The SSH agent is not running in this session.",
			Fix: "Start an agent in your terminal (eval $(ssh-agent)) or use your desktop's key manager, " +
				"then use \"Add to agent\" on the key screen. Note: testing works without the agent too — this only matters if your key has a passphrase.",
		})

	case strings.Contains(low, "load pubkey") && !strings.Contains(low, "successfully"):
		out = append(out, Diagnosis{
			Cause: "One of your key files looks invalid or corrupted.",
			Fix:   "Regenerate the key from the home screen, or check ~/.ssh for leftover files with similar names.",
		})

	case strings.Contains(low, "kex_exchange_identification"), strings.Contains(low, "banner exchange"):
		out = append(out, Diagnosis{
			Cause: "The connection was cut before login could start.",
			Fix:   "This is almost always a firewall or VPN interfering. Disconnect any VPN and retry, or switch networks.",
		})
	}

	if len(out) == 0 {
		out = append(out, Diagnosis{
			Cause: "Authentication did not succeed.",
			Fix: "Make sure the public key was added on the service's website for THIS machine's key, " +
				"then retry. Use \"Copy diagnostics\" below to get help if it still fails.",
		})
	}
	return out
}

func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// hostNameFrom best-effort extracts a hostname from ssh error output.
func hostNameFrom(output string) string {
	for _, marker := range []string{"connect to host ", "Could not resolve hostname ", "to host "} {
		if i := strings.Index(output, marker); i >= 0 {
			rest := output[i+len(marker):]
			if j := strings.IndexAny(rest, " ,:"); j > 0 {
				return rest[:j]
			}
		}
	}
	return "the Git service"
}
