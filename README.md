# KeySmith

A minimal desktop application that helps you generate, manage, and verify SSH keys with a guided UI.

## Features

- Guided setup checklist in-app (generate, add to agent, copy/add to host, test)
- Key management sidebar (view, select, refresh, delete)
- Generates SSH keys with algorithm choice: Ed25519, RSA, or ECDSA
- Optional key passphrase support during key creation
- Optional key comment and overwrite support
- Shows selected key fingerprint for quick verification
- Adds selected private key to SSH agent
- Starts/checks SSH agent (best effort, with platform-aware messaging)
- Copies selected public key to clipboard
- Opens GitHub and Bitbucket SSH settings pages directly
- Tests SSH connectivity to GitHub and Bitbucket using selected identity
- Activity log for every operation
- Remembers key workflow state across app restarts (added/copied/tested/agent markers)

## Requirements

- Go 1.27+ (for the native app)
- OpenSSH tools available on your system (`ssh`, `ssh-keygen`, `ssh-add`)
- Linux GUI builds additionally need OpenGL/GLFW dev packages (`libgl1-mesa-dev xorg-dev`)

## Quick start (development)

1. Build and run the desktop GUI:

   - `go run ./cmd/keysmith`

2. Or run the terminal UI:

   - `go run -tags tui ./cmd/keysmith --tui`

3. On Linux, source the toolchain env first if Go/GL live outside the system paths:

   - `. ./env.sh`

## Install

The fastest way to get the app is via npm (downloads a prebuilt binary on
first run), or grab a binary directly from
[Releases](https://github.com/Leptons1618/keysmith/releases):

```sh
npm install -g keysmith
keysmith          # desktop GUI
keysmith --tui    # terminal UI
```

## How to use the app

1. Click `Start / Check SSH Agent`
2. Enter key details and click `Generate Key`
3. Select the key from the left sidebar
4. Click `Add Key to SSH Agent`
5. Click `Copy Public Key`
6. Open your Git host SSH settings page and paste the key
7. Click `Run SSH Test`

The `Guided Steps` panel shows what is done and what is still pending for the selected key.

For a more detailed explanation, see:

- docs/SETUP_GUIDE.md

## Building a native executable

- Desktop GUI (current platform): `go build ./cmd/keysmith`
- Terminal UI: `CGO_ENABLED=0 go build -tags tui ./cmd/keysmith`

Output is a single binary in the repo root.

## Creating releases

This repository includes GitHub Actions workflows that build executables for
Windows, macOS, and Linux (GUI and TUI) plus an npm package when you push a tag.

1. Create a tag (recommended format: `vMAJOR.MINOR.PATCH`)
2. Push the tag

Example:

- `git tag v1.0.0`
- `git push origin v1.0.0`

The workflow will attach GUI binaries, TUI binaries, and the npm tarball to
the GitHub Release (and publish the npm package when `NPM_TOKEN` is set).

For details, see:

- docs/RELEASING.md

## Troubleshooting

- If starting the SSH agent fails on Windows, ensure the `OpenSSH Authentication Agent` service exists and is not disabled.
- If `ssh-add` fails, run `ssh-add -l` in a terminal to inspect agent state.
- If the connection test fails, confirm that the public key was added to your Git host account.
