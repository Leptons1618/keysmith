# Contributing

## Development setup

- Go 1.27+
- OpenSSH tools available in `PATH`: `ssh`, `ssh-keygen`, and `ssh-add`
- Linux GUI builds require the OpenGL/GLFW and Wayland development packages listed in `.github/workflows/ci.yml`

Install Go dependencies and run the focused checks:

```sh
go test -tags gui ./...
go test -tags tui ./...
go vet -tags gui ./...
go vet -tags tui ./...
node --test cmd/keysmith/launcher.test.js
```

Run the desktop GUI:

```sh
go run ./cmd/keysmith
```

Run the terminal UI:

```sh
go run -tags tui ./cmd/keysmith --tui
```

The GUI and TUI are separate release binaries selected by build tags. Each binary accepts only its own frontend flag; use the npm `keysmith` or `keysmith-tui` command when you need automatic platform asset selection.

## Pull requests

- Keep changes focused and small.
- Prefer changes that preserve the shared `internal/core` workflow and keep GUI/TUI behavior aligned.
- Add regression tests for behavior, boundaries, transitions, and errors.
- Do not add a second key-management implementation or bypass `internal/core` from a frontend.
- Avoid introducing platform-specific behavior unless the release matrix requires it.
