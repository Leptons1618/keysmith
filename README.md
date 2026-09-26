# KeySmith

KeySmith is a Go desktop application for generating, managing, and verifying SSH keys through a guided GUI or terminal UI. The shared core operates on `~/.ssh` and shells out to OpenSSH tools; the private key never leaves the machine.

## Features

- Guided setup flow: check the agent, generate a key, copy its public half, add it to a Git service, and test the selected identity.
- Existing-key management: view fingerprints and workflow markers, copy public keys, add keys to the SSH agent, test connections, and delete key pairs.
- Ed25519, RSA-4096, and ECDSA key generation with optional comment, passphrase, and explicit overwrite.
- GitHub, GitLab, and Bitbucket setup instructions and connection tests.
- Port 443 fallback for services when the standard SSH port is blocked.
- Failure diagnosis that names the most likely cause and the exact fix, instead of leaving you with raw SSH output.
- An SSH agent panel that reports what the agent is actually holding, with fingerprints.
- Per-key algorithm and fingerprint, plus workflow markers for copied, agent-loaded, used, and tested keys.
- Light and dark themes that follow your system preference.
- TUI: incremental key filtering, a shortcut overlay on `?`, and a step tracker in the header.
- TUI support for Windows, macOS, and Linux through the npm launcher.

## The desktop app

The GUI is a normal application window, not a chain of dialogs:

- A **navigation rail** on the left holds every destination, plus quick actions
  and a live count of your keys, so nothing is more than one click away.
- A **page header** shows the title, a one-line explanation, and — while you are
  in the setup flow — which of the three stages you are on.
- A **status strip** along the bottom carries the result of the last action,
  coloured by outcome. It replaces the error dialogs that used to interrupt you.
- The **Keys** page shows every key in `~/.ssh` with its algorithm, fingerprint,
  and progress (copied, loaded in the agent, verified), and every action for the
  selected key.
- The **SSH agent** page reports whether this session can reach the agent, and
  lists the keys it holds.

The **New key** page keeps a live summary of what will be written while you
type, and warns you when a key of that name already exists.

## The terminal app

The TUI mirrors the same flow with a step tracker in the header and the same
palette as the GUI. Two things are worth knowing:

- Press `?` anywhere for the shortcut overlay. It shows the global keys and the
  ones specific to the screen you are on, and `t` cycles through the sections.
- Press `/` on the key wall to filter keys as you type. `esc` clears the filter
  and restores the full list.


## Requirements

- Go 1.27+ for local builds.
- OpenSSH tools in `PATH`: `ssh`, `ssh-keygen`, and `ssh-add`.
- Linux GUI builds additionally need the OpenGL/GLFW and Wayland development packages used by CI.
- TUI clipboard actions need `pbcopy` on macOS, `clip.exe` on Windows, or one of `wl-copy`, `xclip`, or `xsel` on Linux.
- TUI browser actions use the platform opener (`open`, `rundll32`, or `xdg-open`).

## Quick start

Desktop GUI:

```sh
go run ./cmd/keysmith
```

Terminal UI:

```sh
go run -tags tui ./cmd/keysmith --tui
```

On Linux, source `env.sh` first if the Go/GL toolchain is outside the system paths:

```sh
. ./env.sh
```

## Install from npm

The npm launcher selects the released GUI or TUI asset for the current platform and caches it after the first run:

```sh
npm install -g keysmith
keysmith          # desktop GUI
keysmith --tui    # terminal UI
```

The default GUI release is available for Linux amd64, macOS arm64, and Windows amd64. The TUI is cross-compiled for amd64 and arm64 on Linux, macOS, and Windows. Unsupported combinations fail before a download.

## Build

```sh
# Desktop GUI for the current platform
go build -tags gui -trimpath ./cmd/keysmith

# Terminal UI
CGO_ENABLED=0 go build -tags tui -trimpath ./cmd/keysmith
```

The GUI and TUI are deliberately separate build artifacts because Fyne links native desktop libraries while the TUI remains pure Go.

## Workflow

1. Open the **SSH agent** page (GUI) or choose **Check SSH agent** (TUI) to see whether this session can reach the agent and what it holds.
2. Generate a key, or open the **Keys** page to manage an existing one.
3. Copy the public key and add the private key to the agent if needed.
4. Open the selected service's SSH-key page, paste the public key, and return to KeySmith.
5. Run the connection test. The result screen diagnoses common network, permission, and authentication failures.

The setup guide contains the detailed host flow:

- [Setup guide](docs/SETUP_GUIDE.md)

## Releases

GitHub Actions validates the exact tag, runs GUI and TUI quality gates, builds all supported artifacts, inspects the npm tarball, creates the GitHub Release, and publishes npm only when `NPM_TOKEN` is configured. See [RELEASING.md](docs/RELEASING.md) for the release matrix and manual verification commands.

## Troubleshooting

- On Windows, ensure the OpenSSH Authentication Agent service exists and is enabled.
- If `ssh-add` fails, run `ssh-add -l` in a terminal to inspect agent state.
- If a connection test fails, confirm the public key is saved in the selected host account and that the intended private key is loaded.
- If the TUI cannot copy, install the clipboard utility for your platform; see Requirements.
