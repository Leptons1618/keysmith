# SSH Key Setup (GUI)

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

- Python 3.12+
- OpenSSH tools available on your system (`ssh`, `ssh-keygen`, `ssh-add`)

## Quick start (development)

1. Create and activate a virtual environment

   Windows (PowerShell):

   - `python -m venv .venv`
   - `.\.venv\Scripts\Activate.ps1`

2. Install dependencies

   - `pip install -r requirements.txt`

3. Run the app

   - `python main.py`

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

This project uses PyInstaller for packaging.

- Install PyInstaller: `pip install pyinstaller`
- Build: `pyinstaller --noconfirm --clean --onefile --windowed --name ssh-key-setup main.py`
- Output: `dist/` contains the packaged executable

## Creating releases

This repository includes a GitHub Actions workflow that builds executables for Windows, macOS, and Linux when you push a tag.

1. Create a tag (recommended format: `vMAJOR.MINOR.PATCH`)
2. Push the tag

Example:

- `git tag v1.0.0`
- `git push origin v1.0.0`

The workflow will attach platform-specific zip files to the GitHub Release.

For details, see:

- docs/RELEASING.md

## Troubleshooting

- If starting the SSH agent fails on Windows, ensure the `OpenSSH Authentication Agent` service exists and is not disabled.
- If `ssh-add` fails, run `ssh-add -l` in a terminal to inspect agent state.
- If the connection test fails, confirm that the public key was added to your Git host account.
