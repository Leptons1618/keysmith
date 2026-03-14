# Setup Guide

This guide describes the recommended SSH setup flow used by the application.

## Step 1: Start/check the SSH agent

The SSH agent keeps private keys in memory so SSH can authenticate without repeatedly prompting you.

- Windows: click `Start / Check SSH Agent` and the app attempts to start the built-in `ssh-agent` service.
- macOS/Linux: click `Start / Check SSH Agent` to verify agent availability for the current session.

If the agent cannot be started, ensure OpenSSH is installed and enabled on your system.

## Step 2: Generate an SSH key

The app can generate `ed25519`, `rsa`, or `ecdsa` keys. The default key path is:

- Private key: `~/.ssh/id_ed25519`
- Public key: `~/.ssh/id_ed25519.pub`

If the key already exists, the app will not overwrite it unless you explicitly choose the overwrite option.

You can also set an optional passphrase when generating a key.

## Step 3: Add the selected key to the agent

Select a key in the sidebar, then click `Add Key to SSH Agent`.

The app runs `ssh-add` to load the selected private key into the agent.

If you used a passphrase, `ssh-add` may prompt for it depending on your environment.

If this fails:

- Verify that `ssh-add` exists in your PATH.
- In a terminal, run `ssh-add -l` to see whether the agent has identities.

## Step 4: Add the public key to your Git host

Copy the public key and add it to your Git host account. The app can open these pages directly from the UI.

Common URLs:

- GitHub: https://github.com/settings/ssh/new
- Bitbucket: https://bitbucket.org/account/settings/ssh-keys/

## Step 5: Test connectivity

The app tests:

- `ssh -T git@github.com`
- `ssh -T git@bitbucket.org`

If the test fails:

- Confirm the public key was saved to your Git host.
- Confirm the agent is running and the key is loaded.
- Confirm you are using the expected key path.

The app also shows your selected key fingerprint so you can verify the exact key identity.
