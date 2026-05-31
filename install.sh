#!/usr/bin/env bash
#
# antibot installer — downloads a prebuilt binary (no dependencies needed).
#
#   curl -fsSL https://raw.githubusercontent.com/albinstman/antibot-print/main/install.sh | bash
#
# Detects your OS/arch, downloads the matching binary from the latest release,
# verifies its SHA-256 checksum, and installs it onto your PATH. Override defaults:
#
#   ANTIBOT_BIN_DIR=/usr/local/bin   # install location (default: ~/.local/bin)
#   ANTIBOT_REF=latest               # release tag to install (default: latest)
#   ANTIBOT_REPO=owner/name          # source repo (default: albinstman/antibot-print)
#
set -euo pipefail

REPO="${ANTIBOT_REPO:-albinstman/antibot-print}"
REF="${ANTIBOT_REF:-latest}"
BIN_DIR="${ANTIBOT_BIN_DIR:-$HOME/.local/bin}"

if [ -t 2 ]; then
  B=$'\033[1m'; G=$'\033[32m'; Y=$'\033[33m'; R=$'\033[31m'; X=$'\033[0m'
else
  B=""; G=""; Y=""; R=""; X=""
fi
info() { printf '%s==>%s %s\n' "$B" "$X" "$*" >&2; }
ok()   { printf '%s\xe2\x9c\x93%s %s\n' "$G" "$X" "$*" >&2; }
warn() { printf '%s!%s %s\n' "$Y" "$X" "$*" >&2; }
die()  { printf '%s\xe2\x9c\x97 %s%s\n' "$R" "$*" "$X" >&2; exit 1; }

# --- pick the right asset ----------------------------------------------------
os="$(uname -s)"
case "$os" in
  Linux)  os=linux ;;
  Darwin) os=darwin ;;
  *) die "unsupported OS '$os'. Windows users: download antibot-windows-amd64.exe from
     https://github.com/${REPO}/releases/${REF}" ;;
esac
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64)  arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) die "unsupported architecture '$arch'." ;;
esac
# Apple Silicon: an x86_64 shell running under Rosetta 2 should still get arm64.
if [ "$os" = "darwin" ] && [ "$arch" = "amd64" ] \
   && [ "$(sysctl -n sysctl.proc_translated 2>/dev/null)" = "1" ]; then
  arch=arm64
fi
asset="antibot-${os}-${arch}"

if [ "$REF" = "latest" ]; then
  base="https://github.com/${REPO}/releases/latest/download"
else
  base="https://github.com/${REPO}/releases/download/${REF}"
fi

# --- downloader --------------------------------------------------------------
if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO "$2" "$1"; }
else
  die "need either curl or wget to download."
fi

tmp="$(mktemp -d "${TMPDIR:-/tmp}/antibot.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT

info "Downloading ${asset} (${REPO}@${REF})"
fetch "${base}/${asset}"      "$tmp/$asset"      || die "download failed: ${base}/${asset}"
fetch "${base}/SHA256SUMS"    "$tmp/SHA256SUMS"  || die "download failed: ${base}/SHA256SUMS"
[ -s "$tmp/$asset" ] || die "downloaded binary is empty."

# --- verify checksum ---------------------------------------------------------
info "Verifying checksum"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$asset" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')"
else
  die "need sha256sum or shasum to verify the download."
fi
expected="$(awk -v f="$asset" '$2 == f || $2 == "*"f {print $1}' "$tmp/SHA256SUMS")"
[ -n "$expected" ] || die "no checksum for $asset in SHA256SUMS."
[ "$actual" = "$expected" ] || die "checksum mismatch for $asset (expected $expected, got $actual)."

# --- self-test, then install -------------------------------------------------
chmod +x "$tmp/$asset"
info "Verifying it runs"
printf 'HTTP/1.1 403\r\nSet-Cookie: __cf_bm=x; path=/\r\n\r\n' | "$tmp/$asset" 2>/dev/null \
  | grep -qx cloudflare || die "self-test failed (the binary did not detect a Cloudflare response)."

mkdir -p "$BIN_DIR" || die "cannot create $BIN_DIR"
dest="$BIN_DIR/antibot"
cp "$tmp/$asset" "$dest" && chmod 0755 "$dest" || die "cannot write $dest"
ok "Installed $("$dest" --version) -> $dest"

case ":$PATH:" in
  *":$BIN_DIR:"*) : ;;
  *) warn "$BIN_DIR is not on your PATH. Add to your shell profile:"
     printf '\n    export PATH="%s:$PATH"\n' "$BIN_DIR" >&2 ;;
esac

# The command was renamed antibot-print -> antibot; flag a leftover old binary.
if [ -e "$BIN_DIR/antibot-print" ]; then
  warn "An old 'antibot-print' binary remains in $BIN_DIR — the command is now 'antibot'."
  printf "    Remove the stale binary with: rm '%s/antibot-print'\n" "$BIN_DIR" >&2
fi

cat >&2 <<EOF

Try it:
    curl -isS https://example.com | antibot
EOF
