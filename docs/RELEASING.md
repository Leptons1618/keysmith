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

Use release tags in strict SemVer form:

- `vMAJOR.MINOR.PATCH` (for example, `v1.2.0`)
- `vMAJOR.MINOR.PATCH-PRERELEASE` (for example, `v1.2.0-rc.1`)

Leading zeroes are not allowed in the numeric version or numeric prerelease
identifiers. Build metadata (`+build`) is not accepted because npm package
versions cannot publish it unchanged.

## Create a release

1. Ensure your working tree is clean and you are on the commit you want to release.
2. Create a tag:

   - `git tag v1.0.0`

3. Push the tag:

   - `git push origin v1.0.0`

You can also run the workflow manually from the GitHub Actions tab
(**Release -> Run workflow**) and provide the name of an existing tag. The
workflow always checks out that exact tag, not the branch selected in the
Actions UI.

## What happens in CI

The workflow in `.github/workflows/release.yml` will:

1. Validate the tag is a supported strict SemVer release tag and check out
   that exact tag.
2. Run GUI and TUI tests and vet checks as a quality gate before any artifact
   job starts.
3. Build the desktop GUI natively on Ubuntu, macOS, and Windows runners.
4. Cross-compile the TUI for six GOOS/GOARCH combinations.
5. Stage and inspect the npm package (`package.release.json` plus the
   `bin/keysmith.js` launcher) with the version filled in from the tag.
6. Verify the complete set of non-empty release asset names, then create the
   GitHub Release with all binaries and the npm tarball attached.
7. Publish the tarball to npm as `keysmith` after the GitHub Release exists,
   but only if the `NPM_TOKEN` repository secret is set. A configured publish
   failure fails the workflow; only a missing token skips npm publishing.

Stable versions publish under npm's `latest` tag. Prerelease versions publish
under `next` so they do not replace the latest stable installation.

Only one release workflow runs for a given tag at a time. A queued run waits
for the active run rather than cancelling it.

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
go build -tags gui -trimpath -ldflags "-X main.version=$(git describe --tags --always)" ./cmd/keysmith

# Terminal UI (any platform)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags tui -o /tmp/skm-tui ./cmd/keysmith

# npm package preview
rm -rf pkg out && mkdir -p pkg/bin out
cp package.release.json pkg/package.json
cp bin/keysmith.js pkg/bin/keysmith.js
chmod +x pkg/bin/keysmith.js
sed -i 's|0.0.0-PLACEHOLDER|1.0.0|g' pkg/package.json
(cd pkg && npm pack --pack-destination ../out)
```

## Notes

- Fyne GUI binaries are not code-signed. On macOS, Gatekeeper may warn users;
  on Windows, SmartScreen may flag the executable. Add code signing as a
  separate step if needed.
- The TUI binaries are static (CGO disabled), so they run on minimal systems.
