#!/usr/bin/env bash
set -euo pipefail
REPO="ivankuzyshyn/dotfiles"
PREFIX="${PREFIX:-$HOME/.local/bin}"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in x86_64) ARCH=amd64 ;; aarch64|arm64) ARCH=arm64 ;; esac
VER="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep -m1 '"tag_name"' | cut -d'"' -f4)"
TARBALL="dot_${VER#v}_${OS}_${ARCH}.tar.gz"
CHECKSUMS_URL="https://github.com/$REPO/releases/download/$VER/checksums.txt"
TARBALL_URL="https://github.com/$REPO/releases/download/$VER/$TARBALL"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
curl -fsSL "$TARBALL_URL" -o "$TMP/$TARBALL"
curl -fsSL "$CHECKSUMS_URL" -o "$TMP/checksums.txt"
(cd "$TMP" && grep " $TARBALL\$" checksums.txt | shasum -a 256 -c -)
tar -xzf "$TMP/$TARBALL" -C "$TMP"
mkdir -p "$PREFIX"
install -m 0755 "$TMP/dot" "$PREFIX/dot"
echo "Installed dot to $PREFIX/dot"
case ":$PATH:" in *":$PREFIX:"*) ;; *) echo "Add to PATH: export PATH=\"$PREFIX:\$PATH\"" ;; esac
