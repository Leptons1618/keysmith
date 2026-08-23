# Releasing

This repository uses GitHub Actions to build native executables for Windows,
macOS, and Linux, plus an npm package, and publishes them to GitHub Releases
(and npm) when you push a tag.

## What gets built

| Artifact | Frontend | Platforms | Notes |
| --- | --- | --- | --- |
| `keysmith-<tag>-<os>-<arch>` | Desktop GUI (Fyne) | linux-amd64, darwin-arm64, windows-amd64 | Built natively on each OS; needs OpenGL/GLFW dev packages on Linux |
| `keysmith-tui-<tag>-<os>-<arch>` | Terminal UI (Bubbletea) | linux amd64/arm64, darwin amd64/arm64, windows amd64/arm64 | Pure Go (`-tags tui`), cross-compiled, no C toolchain |
| `keysmith-<version>.tgz` | npm launcher | any (Node 18+) | Downloads the right binary from GitHub Releases on first run |

## Versioning

Use tags in the format:

- `vMAJOR.MINOR.PATCH` (example: `v1.2.0`)

## Create a release

1. Ensure your working tree is clean and you are on the commit you want to release.
2. Create a tag:

   - `git tag v1.0.0`

3. Push the tag:

   - `git push origin v1.0.0`

You can also run the workflow manually from the GitHub Actions tab
(**Release -> Run workflow**) and provide the tag input.

## What happens in CI

The workflow in `.github/workflows/release.yml` will:

1. Validate the tag looks like `vMAJOR.MINOR.PATCH`.
2. Build the desktop GUI natively on Ubuntu, macOS, and Windows runners.
3. Cross-compile the TUI for six GOOS/GOARCH combinations.
4. Stage the npm package (`package.release.json` + `bin/` launcher) with the
   version filled in from the tag.
5. Create a GitHub Release with all binaries and the tarball attached.
6. Publish the package to npm as `keysmith` **only if** the
   `NPM_TOKEN` repository secret is set; otherwise it warns and skips.

## npm setup (one-time)

To enable publishing:

1. Create an npm automation token (<https://www.npmjs.com/settings/your-user/tokens>)
   - "Automation" tokens skip two-factor prompts in CI.
2. Add it as a repository secret named `NPM_TOKEN`
   (Settings -> Secrets and variables -> Actions).

The npm name is `keysmith`; users then run:

```sh
npm install -g keysmith
keysmith          # desktop GUI
keysmith --tui    # terminal UI
```

## Local dry run

Build what CI builds:

```sh
# Desktop GUI (current platform)
go build -trimpath -ldflags "-X main.version=$(git describe --tags --always)" ./cmd/keysmith

# Terminal UI (any platform)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags tui -o /tmp/skm-tui ./cmd/keysmith

# npm package preview
mkdir -p pkg/bin && cp package.release.json pkg/package.json && cp bin/* pkg/bin/
(cd pkg && npm pack)
```

## Notes

- Fyne GUI binaries are not code-signed. On macOS, Gatekeeper may warn users;
  on Windows, SmartScreen may flag the executable. Add code signing as a
  separate step if needed.
- The TUI binaries are static (CGO disabled), so they run on minimal systems.
- The Python app (`main.py`) is legacy and is not part of the release pipeline.
