# SSH Key Setup (GUI)

A small desktop application that helps you generate an SSH key, load it into an SSH agent, and verify connectivity to common Git hosts.

## Features

- Guided setup (step-by-step)
- Generates an Ed25519 SSH key
- Adds the key to the SSH agent
- Copies the public key to your clipboard
- Tests SSH connectivity to GitHub and Bitbucket

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

## How the setup works

1. Start the SSH agent
2. Generate an SSH key (Ed25519)
3. Add the key to the agent
4. Copy the public key and add it to your Git host
5. Test the SSH connection

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
