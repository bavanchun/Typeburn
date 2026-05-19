#!/bin/sh
# Typeburn installer — POSIX sh, no sudo, no Go toolchain.
#
#   curl -fsSL https://raw.githubusercontent.com/bavanchun/Typeburn/main/install.sh | sh
#
# Trust boundary (read this before piping any script to a shell):
#   The sha256 verification below defends against a corrupted or
#   man-in-the-middled DOWNLOAD. It does NOT make `curl … | sh` "safe": the
#   script, the archive, and checksums.txt all come from the same GitHub
#   release, so a compromised release would be self-consistent. checksums.txt
#   is unsigned. If that boundary matters to you, use the non-piped audit path
#   in the README (download, read this script, verify, then run).
#
# What it does: detect os/arch, resolve the latest NON-prerelease tag, download
# the matching release archive + checksums.txt, verify sha256, validate the
# archive member, then atomically install `typeburn` into ~/.local/bin.
#
# Env (the TYPEBURN_* / *_UNAME_* knobs exist for the offline test harness):
#   VERSION              pin a tag (e.g. v1.5.0); the API "prerelease":true
#                        check is skipped (operator intent) but the string
#                        guard still rejects -rc/-test/-beta/… tags
#   BIN_DIR              install dir (default: ~/.local/bin)
#   TYPEBURN_API         GitHub API base
#   TYPEBURN_BASE_URL    release-download base
#   TYPEBURN_LATEST_PATH API path for "latest" (default: releases/latest)
#   TYPEBURN_UNAME_S/M   override uname -s / -m
set -eu

REPO="bavanchun/Typeburn"
API="${TYPEBURN_API:-https://api.github.com/repos/$REPO}"
BASE_URL="${TYPEBURN_BASE_URL:-https://github.com/$REPO/releases/download}"
LATEST_PATH="${TYPEBURN_LATEST_PATH:-releases/latest}"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
BIN_NAME="typeburn"

die() { printf 'install.sh: %s\n' "$*" >&2; exit 1; }
info() { printf '==> %s\n' "$*"; }
warn() { printf 'install.sh: warning: %s\n' "$*" >&2; }

# fetch <url> <dest> : explicit failure (POSIX sh has no `pipefail`; never
# rely on a pipe's exit status — check every download by hand).
fetch() {
  _u="$1"; _d="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "$_d" "$_u" || return 1
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$_d" "$_u" || return 1
  else
    die "need curl or wget"
  fi
  [ -s "$_d" ] || return 1   # reject empty/truncated-to-zero downloads
}

is_prerelease() {
  # Defence-in-depth against a poisoned `releases/latest` (a disposable
  # -rc dry-run tag) and against an explicit VERSION=-rc override.
  case "$1" in
    *-rc*|*-RC*|*-test*|*-alpha*|*-beta*|*-pre*|*-dev*|*-snapshot*|*SNAPSHOT*)
      return 0 ;;
    *) return 1 ;;
  esac
}

# --- platform -------------------------------------------------------------
uname_s="${TYPEBURN_UNAME_S:-$(uname -s)}"
uname_m="${TYPEBURN_UNAME_M:-$(uname -m)}"
case "$uname_s" in
  Linux)  os="linux" ;;
  Darwin) os="darwin" ;;
  *) die "unsupported OS '$uname_s' — install.sh covers linux and darwin only. Windows: download the .zip from the releases page manually." ;;
esac
case "$uname_m" in
  x86_64|amd64)  arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) die "unsupported architecture '$uname_m' — supported: x86_64/amd64, arm64/aarch64." ;;
esac

# --- resolve version ------------------------------------------------------
tmp="$(mktemp -d)"
chmod 700 "$tmp"
stage=""
cleanup() {
  rm -rf "$tmp"
  if [ -n "$stage" ]; then
    rm -f "$stage" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM HUP

if [ -n "${VERSION:-}" ]; then
  tag="$VERSION"
else
  info "resolving latest release"
  fetch "$API/$LATEST_PATH" "$tmp/latest.json" \
    || die "could not query the releases API (set VERSION=vX.Y.Z to bypass, e.g. on rate-limit)"
  if grep -q '"prerelease"[[:space:]]*:[[:space:]]*true' "$tmp/latest.json"; then
    die "refusing: the resolved 'latest' release is flagged prerelease"
  fi
  tag="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$tmp/latest.json" | head -1)"
fi
[ -n "$tag" ] || die "could not determine a release tag"
if is_prerelease "$tag"; then
  die "refusing prerelease/test tag '$tag' (install.sh installs stable releases only)"
fi

ver="${tag#v}"   # GoReleaser strips the leading v from archive names
archive="typeburn_${ver}_${os}_${arch}.tar.gz"
info "installing $tag ($os/$arch)"

# --- download + verify ----------------------------------------------------
fetch "$BASE_URL/$tag/$archive" "$tmp/$archive" \
  || die "download failed: $BASE_URL/$tag/$archive"
fetch "$BASE_URL/$tag/checksums.txt" "$tmp/checksums.txt" \
  || die "download failed: checksums.txt for $tag"

expected="$(awk -v f="$archive" '$2==f {print $1}' "$tmp/checksums.txt" | head -1)"
[ -n "$expected" ] || die "no checksum entry for $archive in checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$archive" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')"
else
  die "need sha256sum or shasum to verify the download"
fi
[ "$expected" = "$actual" ] \
  || die "checksum mismatch for $archive (expected $expected, got $actual) — aborting, nothing installed"
info "sha256 verified"

# --- extract + validate member -------------------------------------------
# Reject absolute paths and `..` path COMPONENTS before extracting anything
# (precise: a benign filename like `foo..bar` must not trip this — the
# literal-member extraction below is the primary control regardless).
tar -tzf "$tmp/$archive" | while IFS= read -r entry; do
  case "$entry" in
    /*|../*|*/../*|*/..|..) echo "BADPATH" ; break ;;
  esac
done | grep -q BADPATH && die "archive contains an unsafe path — aborting"

mkdir -p "$tmp/x"
tar -xzf "$tmp/$archive" -C "$tmp/x" "$BIN_NAME" 2>/dev/null \
  || die "archive does not contain a '$BIN_NAME' entry"
[ -e "$tmp/x/$BIN_NAME" ] || die "extracted '$BIN_NAME' missing"
[ ! -L "$tmp/x/$BIN_NAME" ] || die "refusing: '$BIN_NAME' in the archive is a symlink"
[ -f "$tmp/x/$BIN_NAME" ] || die "refusing: '$BIN_NAME' in the archive is not a regular file"

# --- atomic install -------------------------------------------------------
mkdir -p "$BIN_DIR"
stage="$BIN_DIR/.$BIN_NAME.$$.tmp"   # same filesystem as the final path
cp "$tmp/x/$BIN_NAME" "$stage"
chmod 0755 "$stage"
mv -f "$stage" "$BIN_DIR/$BIN_NAME"  # rename = atomic; prior binary intact until here
stage=""
info "installed $BIN_DIR/$BIN_NAME"

# --- PATH membership + precedence ----------------------------------------
case ":$PATH:" in
  *":$BIN_DIR:"*) on_path=1 ;;
  *) on_path=0 ;;
esac
if [ "$on_path" -eq 0 ]; then
  warn "$BIN_DIR is not on your PATH. Add it:"
  # Literal $PATH is intentional — this is a copy-paste line for the user's shell.
  # shellcheck disable=SC2016
  printf '    export PATH="%s:$PATH"\n' "$BIN_DIR" >&2
else
  resolved="$(command -v "$BIN_NAME" 2>/dev/null || true)"
  if [ -n "$resolved" ] && [ "$resolved" != "$BIN_DIR/$BIN_NAME" ]; then
    warn "another '$BIN_NAME' takes PATH precedence: $resolved (just installed: $BIN_DIR/$BIN_NAME)"
  fi
fi

info "done — run: $BIN_NAME --version"
