# Build environment for the userland Go + Fyne toolchain (no sudo required).
# Source this before running go commands:  . ./env.sh
export PATH="$HOME/.local/go/bin:$PATH"
export LIBRARY_PATH="$HOME/.local/lib${LIBRARY_PATH:+:$LIBRARY_PATH}"
export PKG_CONFIG_PATH="$HOME/.local/lib/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
export CGO_CFLAGS="-I$HOME/.local/include"
