# Releasing

This repository uses GitHub Actions to build native executables and publish them to GitHub Releases.

## Versioning

Use tags in the format:

- `vMAJOR.MINOR.PATCH` (example: `v1.2.0`)

## Create a release

1. Ensure your working tree is clean and you are on the commit you want to release.
2. Create a tag:

   - `git tag v1.0.0`

3. Push the tag:

   - `git push origin v1.0.0`

## What happens in CI

The workflow in `.github/workflows/release.yml` will:

- Build a standalone executable using PyInstaller on:
  - Windows
  - macOS
  - Linux
- Zip each build output with a platform-specific file name
- Create (or update) a GitHub Release for the tag and attach the zip files

## Notes

- PyInstaller output is not code-signed. On macOS, Gatekeeper may warn users.
- If you need signed binaries, add code signing as a separate step.
