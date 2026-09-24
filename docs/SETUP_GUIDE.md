# Setup Guide

This guide describes the SSH setup flow used by the Go GUI and TUI.

## Step 1: Check the SSH agent

The SSH agent keeps private keys in memory so SSH can authenticate without repeatedly prompting you.

- Windows: choose **Check SSH agent**; KeySmith asks the built-in `ssh-agent` service to start when needed.
- macOS/Linux: choose **Check SSH agent** to verify the agent for the current session.

If the agent cannot be reached, ensure OpenSSH is installed and enabled on your system.

## Step 2: Generate an SSH key

KeySmith supports `ed25519`, `rsa`, and `ecdsa` keys. The default key name is `id_ed25519`; the pair is stored under `~/.ssh` as:

- Private key: `~/.ssh/id_ed25519`
- Public key: `~/.ssh/id_ed25519.pub`

Existing keys are not overwritten unless the overwrite option is selected. A passphrase is optional during generation.

## Step 3: Add the selected key to the agent

Open **Manage existing keys**, select a key, and choose **Add to agent**. The app runs `ssh-add` for that private key.

If you used a passphrase, `ssh-add` may prompt for it depending on your environment. The TUI runs this operation asynchronously, so a passphrase-protected key may need an agent-compatible environment or a terminal-based `ssh-add` first.

If this fails:

- Verify that `ssh-add` exists in `PATH`.
- In a terminal, run `ssh-add -l` to see whether the agent has identities.

## Step 4: Add the public key to your Git host

Select a service, copy the public key, and open that service’s SSH-key settings page from the app. KeySmith supports:

- GitHub: https://github.com/settings/keys
- GitLab: https://gitlab.com/-/user_settings/ssh_keys
- Bitbucket: https://bitbucket.org/account/settings/ssh-keys/

The `.pub` file is the shareable half. Never paste the private key into a web form.

## Step 5: Test connectivity

Choose the service and run the connection test. KeySmith uses the selected key identity and first tries the standard SSH port. If the failure indicates a blocked network path, it tries the service’s documented port-443 endpoint where available.

A successful test is service-specific: GitHub, GitLab, and Bitbucket use different SSH banners, so the shared result classifier recognizes each provider’s response.

If the test fails:

- Confirm the public key was saved to the selected host account.
- Confirm the agent is running and the key is loaded.
- Confirm the selected key fingerprint matches the key added to the host.
- Check the result screen diagnosis for network, host-key, permission, and agent errors.

KeySmith shows the selected key fingerprint and remembers workflow markers after the app restarts.
