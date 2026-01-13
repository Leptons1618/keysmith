# Setup Guide

This guide describes the recommended SSH setup flow used by the application.

## Step 1: Start the SSH agent

The SSH agent keeps private keys in memory so SSH can authenticate without repeatedly prompting you.

- Windows: the app attempts to start the built-in `ssh-agent` service.
- macOS/Linux: the app attempts to start an agent for the current session.

If the agent cannot be started, ensure OpenSSH is installed and enabled on your system.

## Step 2: Generate an SSH key

The app generates an Ed25519 key pair at:

- Private key: `~/.ssh/id_ed25519`
- Public key: `~/.ssh/id_ed25519.pub`

If the key already exists, the app will not overwrite it unless you explicitly choose the overwrite option.

## Step 3: Add the key to the agent

The app runs `ssh-add` to load the private key into the agent.

If this fails:

- Verify that `ssh-add` exists in your PATH.
- In a terminal, run `ssh-add -l` to see whether the agent has identities.

## Step 4: Add the public key to your Git host

Copy the public key and add it to your Git host account.

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
