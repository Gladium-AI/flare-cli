#!/bin/sh
set -e

REPO="Gladium-AI/flare-cli"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BINARY="flare"

# Detect OS.
OS="$(uname -s)"
case "$OS" in
  Linux*)  GOOS="linux" ;;
  Darwin*) GOOS="darwin" ;;
  *)       echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture.
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  GOARCH="amd64" ;;
  arm64|aarch64)  GOARCH="arm64" ;;
  *)              echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

echo "Detected platform: ${GOOS}/${GOARCH}"

# Get the latest release tag.
if command -v curl >/dev/null 2>&1; then
  TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')"
elif command -v wget >/dev/null 2>&1; then
  TAG="$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')"
else
  echo "Error: curl or wget is required" >&2
  exit 1
fi

if [ -z "$TAG" ]; then
  echo "Error: could not determine latest release" >&2
  exit 1
fi

echo "Latest release: ${TAG}"

ARCHIVE="${BINARY}-${TAG}-${GOOS}-${GOARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"

# Download and extract.
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading ${URL}..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o "${TMPDIR}/${ARCHIVE}"
else
  wget -q "$URL" -O "${TMPDIR}/${ARCHIVE}"
fi

tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"

# Install.
mkdir -p "$INSTALL_DIR"
mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
chmod +x "${INSTALL_DIR}/${BINARY}"

echo "Installed ${BINARY} ${TAG} to ${INSTALL_DIR}/${BINARY}"

# Check if INSTALL_DIR is in PATH.
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo ""
    echo "Add ${INSTALL_DIR} to your PATH:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac
