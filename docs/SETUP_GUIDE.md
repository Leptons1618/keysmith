# Setup Guide

This guide describes the SSH setup flow used by the Go GUI and TUI. The three
stages are the same on both frontends, and the app always shows which one you
are on.

## Step 1: Check the SSH agent

The SSH agent keeps private keys in memory so SSH can authenticate without repeatedly prompting you.

- GUI: open **SSH agent** in the sidebar and choose **Check now**.
- TUI: choose **Check SSH agent** on the home screen, then press `c` to probe.

KeySmith runs `ssh-add -l`, so the answer is real: on Windows it also asks the
built-in `ssh-agent` service to start when needed. The agent screen lists the
keys the agent is actually holding, with their fingerprints, so you can tell at
a glance whether the key you care about is loaded.

If the agent cannot be reached, ensure OpenSSH is installed and enabled on your
system, start it (`eval $(ssh-agent)`), and check again. Testing a key still
works without an agent — the agent only matters when your key has a passphrase.

## Step 2: Generate an SSH key

KeySmith supports `ed25519`, `rsa`, and `ecdsa` keys. The default key name is `id_ed25519`; the pair is stored under `~/.ssh` as:

- Private key: `~/.ssh/id_ed25519`
- Public key: `~/.ssh/id_ed25519.pub`

Existing keys are not overwritten unless the overwrite option is selected. A
passphrase is optional during generation.

The create page shows a live summary of what will be written, and warns before
you create a key whose name is already taken, so a typo cannot silently replace
an existing key.

## Step 3: Add the selected key to the agent

Open the **Keys** page (GUI) or **Manage existing keys** (TUI), select a key,
and add it to the agent. The app runs `ssh-add` for that private key.

If you used a passphrase, `ssh-add` may prompt for it depending on your environment. The TUI runs this operation asynchronously, so a passphrase-protected key may need an agent-compatible environment or a terminal-based `ssh-add` first.

If this fails:

- Verify that `ssh-add` exists in `PATH`.
- In a terminal, run `ssh-add -l` to see whether the agent has identities.
- Re-check the agent page: it lists exactly what is loaded.

## Step 4: Add the public key to your Git host

Select a service, copy the public key, and open that service's SSH-key settings page from the app. KeySmith supports:

- GitHub: https://github.com/settings/keys
- GitLab: https://gitlab.com/-/user_settings/ssh_keys
- Bitbucket: https://bitbucket.org/account/settings/ssh-keys/

The `.pub` file is the shareable half. Never paste the private key into a web form.

The public key is copied to your clipboard as soon as you open the page, and the
instructions screen keeps it visible so you can compare it against what the
website shows you.

## Step 5: Test connectivity

Choose the service and run the connection test. KeySmith uses the selected key identity and first tries the standard SSH port. If the failure indicates a blocked network path, it tries the service's documented port-443 endpoint where available.

A successful test is service-specific: GitHub, GitLab, and Bitbucket use different SSH banners, so the shared result classifier recognizes each provider's response.

If the test fails, the result screen gives you the most likely cause and the
exact fix for each of these:

- The network cannot reach the host (often a blocked port 22).
- The host's identity could not be verified (a stale `known_hosts` entry).
- The service did not accept the key (usually it was never saved on the website).
- The private key could not be read.
- The agent is not running.
- The key files look invalid or corrupted.
- The connection was cut before login could start (firewall or VPN).

KeySmith shows the selected key fingerprint and remembers workflow markers after the app restarts, so the key wall shows each key's progress: copied, loaded in the agent, and verified.

