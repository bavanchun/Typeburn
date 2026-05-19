#!/usr/bin/env bash
# Offline regression harness for install.sh.
#
# No internet: a localhost python http.server stands in for the GitHub API and
# the release-asset download host. install.sh exposes test-only env seams
# (TYPEBURN_API, TYPEBURN_BASE_URL, TYPEBURN_UNAME_S/M, TYPEBURN_LATEST_PATH) so
# every failure mode is exercised deterministically. Each case asserts BOTH the
# exit status and the filesystem: a refused install must write nothing; a failed
# install must leave any prior binary byte-identical.
#
# Subshell-local env per case is the intended isolation, hence the SC2030/SC2031
# suppression — each case must not leak env into the next.
#
# Run: scripts/test-install-sh.sh   (exit 0 = all green)
# shellcheck disable=SC2030,SC2031
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
INSTALL_SH="$REPO_ROOT/install.sh"

WORK="$(mktemp -d)"
SRV_PID=""
PASS=0
FAIL=0

cleanup() {
  if [ -n "$SRV_PID" ]; then
    kill "$SRV_PID" 2>/dev/null || true
  fi
  rm -rf "$WORK"
}
trap cleanup EXIT INT TERM

note() { printf '      %s\n' "$*"; }
ok()   { PASS=$((PASS + 1)); printf 'PASS  %s\n' "$1"; }
bad()  { FAIL=$((FAIL + 1)); printf 'FAIL  %s\n' "$1"; }

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{print $1}'
  else shasum -a 256 "$1" | awk '{print $1}'; fi
}

# --- Fixtures -------------------------------------------------------------
# Real GoReleaser identity (verified from `make snapshot`): archive is
# typeburn_<version-without-v>_<os>_<arch>.tar.gz, member is `typeburn`,
# checksums.txt is GNU `<sha256>␣␣<name>` format.
DOCROOT="$WORK/docroot"
STABLE_TAG="v9.9.9"
STABLE_VER="9.9.9"
mkdir -p "$DOCROOT/api/releases" "$DOCROOT/dl/$STABLE_TAG"

make_archive() { # os arch outdir -> prints archive filename
  local os="$1" arch="$2" out="$3" stage name
  stage="$WORK/stage_${os}_${arch}"
  mkdir -p "$stage"
  printf '#!/bin/sh\necho "typeburn-fake $*"\n' >"$stage/typeburn"
  chmod 755 "$stage/typeburn"
  name="typeburn_${STABLE_VER}_${os}_${arch}.tar.gz"
  tar -czf "$out/$name" -C "$stage" typeburn
  echo "$name"
}

: >"$DOCROOT/dl/$STABLE_TAG/checksums.txt"
for combo in darwin:amd64 darwin:arm64 linux:amd64 linux:arm64; do
  os="${combo%%:*}"; arch="${combo##*:}"
  fn="$(make_archive "$os" "$arch" "$DOCROOT/dl/$STABLE_TAG")"
  printf '%s  %s\n' "$(sha256_of "$DOCROOT/dl/$STABLE_TAG/$fn")" "$fn" \
    >>"$DOCROOT/dl/$STABLE_TAG/checksums.txt"
done

printf '{"tag_name":"%s","prerelease":false}\n' "$STABLE_TAG" \
  >"$DOCROOT/api/releases/latest"
printf '{"tag_name":"%s","prerelease":true}\n' "v0.0.0-rc.test" \
  >"$DOCROOT/api/releases/latest-pre"

# --- Local server ---------------------------------------------------------
( cd "$DOCROOT" && exec python3 -u -m http.server 0 --bind 127.0.0.1 ) \
  >"$WORK/srv.log" 2>&1 &
SRV_PID=$!
PORT=""
for _ in $(seq 1 50); do
  PORT="$(sed -n 's/.* port \([0-9][0-9]*\).*/\1/p' "$WORK/srv.log" | head -1)"
  [ -n "$PORT" ] && break
  sleep 0.1
done
[ -n "$PORT" ] || { echo "server did not start" >&2; cat "$WORK/srv.log" >&2; exit 1; }
BASE="http://127.0.0.1:$PORT"
export TYPEBURN_API="$BASE/api"
export TYPEBURN_BASE_URL="$BASE/dl"

# === (a) os/arch matrix → exact real asset names, happy path =============
for combo in Darwin:x86_64:darwin:amd64 Darwin:arm64:darwin:arm64 \
             Linux:x86_64:linux:amd64 Linux:aarch64:linux:arm64; do
  uS="${combo%%:*}"; rest="${combo#*:}"
  uM="${rest%%:*}"; rest="${rest#*:}"
  goos="${rest%%:*}"; goarch="${rest##*:}"
  bd="$WORK/bin_a_${goos}_${goarch}"
  label="(a) $uS/$uM → typeburn_${STABLE_VER}_${goos}_${goarch}.tar.gz"
  if (
    export BIN_DIR="$bd" TYPEBURN_UNAME_S="$uS" TYPEBURN_UNAME_M="$uM"
    out="$(sh "$INSTALL_SH" 2>&1)" || { echo "$out"; exit 1; }
    [ -x "$bd/typeburn" ] || { echo "binary not installed: $out"; exit 1; }
  ); then ok "$label"; else bad "$label"; fi
done

# === (b) checksum mismatch → nonzero, nothing written ===================
bd="$WORK/bin_b"; CKDIR="$WORK/dl_b/$STABLE_TAG"; mkdir -p "$CKDIR"
cp "$DOCROOT/dl/$STABLE_TAG"/*.tar.gz "$CKDIR/"
sed 's/^[0-9a-f]\{64\}/0000000000000000000000000000000000000000000000000000000000000000/' \
  "$DOCROOT/dl/$STABLE_TAG/checksums.txt" >"$CKDIR/checksums.txt"
if (
  export BIN_DIR="$bd" TYPEBURN_UNAME_S=Linux TYPEBURN_UNAME_M=x86_64 \
         TYPEBURN_BASE_URL="$BASE/dl_b"
  sh "$INSTALL_SH" >/dev/null 2>&1 && exit 1
  [ ! -e "$bd/typeburn" ] || exit 1
); then ok "(b) checksum mismatch refused, nothing written"
else bad "(b) checksum mismatch"; fi

# === (c) unsupported platform → exit 1 + clear message ==================
for plat in Windows:x86_64 FreeBSD:amd64 Linux:mips; do
  uS="${plat%%:*}"; uM="${plat##*:}"; bd="$WORK/bin_c_${uS}_${uM}"
  if (
    export BIN_DIR="$bd" TYPEBURN_UNAME_S="$uS" TYPEBURN_UNAME_M="$uM"
    out="$(sh "$INSTALL_SH" 2>&1)"; rc=$?
    [ "$rc" -eq 1 ] || { echo "rc=$rc"; exit 1; }
    echo "$out" | grep -qi "unsupported" || { echo "no clear msg: $out"; exit 1; }
    [ ! -e "$bd/typeburn" ] || exit 1
  ); then ok "(c) $uS/$uM unsupported → exit 1, clear msg"
  else bad "(c) $uS/$uM unsupported"; fi
done

# === (d) truncated/empty download → abort, no partial install ===========
bd="$WORK/bin_d"; DDIR="$WORK/dl_d/$STABLE_TAG"; mkdir -p "$DDIR"
: >"$DDIR/typeburn_${STABLE_VER}_linux_amd64.tar.gz"
cp "$DOCROOT/dl/$STABLE_TAG/checksums.txt" "$DDIR/checksums.txt"
if (
  export BIN_DIR="$bd" TYPEBURN_UNAME_S=Linux TYPEBURN_UNAME_M=x86_64 \
         TYPEBURN_BASE_URL="$BASE/dl_d"
  sh "$INSTALL_SH" >/dev/null 2>&1 && exit 1
  [ ! -e "$bd/typeburn" ] || exit 1
); then ok "(d) empty/truncated download aborts, nothing written"
else bad "(d) truncated download"; fi

# === (e) BIN_DIR absent (nested) → mkdir -p, still succeeds =============
bd="$WORK/no/such/dir/deep/bin"
if (
  export BIN_DIR="$bd" TYPEBURN_UNAME_S=Linux TYPEBURN_UNAME_M=x86_64
  out="$(sh "$INSTALL_SH" 2>&1)" || { echo "$out"; exit 1; }
  [ -x "$bd/typeburn" ] || exit 1
); then ok "(e) absent nested BIN_DIR created via mkdir -p"
else bad "(e) absent BIN_DIR"; fi

# === (f) prerelease refused — poisoned latest AND VERSION override ======
if (
  export BIN_DIR="$WORK/bin_f1" TYPEBURN_UNAME_S=Linux TYPEBURN_UNAME_M=x86_64 \
         TYPEBURN_LATEST_PATH="releases/latest-pre"
  out="$(sh "$INSTALL_SH" 2>&1)" && exit 1
  echo "$out" | grep -qi "prerelease\|refus" || { echo "no msg: $out"; exit 1; }
  [ ! -e "$WORK/bin_f1/typeburn" ] || exit 1
); then ok "(f) poisoned prerelease latest refused"
else bad "(f) prerelease latest"; fi
if (
  export BIN_DIR="$WORK/bin_f2" TYPEBURN_UNAME_S=Linux TYPEBURN_UNAME_M=x86_64 \
         VERSION="v0.0.0-rc.test"
  sh "$INSTALL_SH" >/dev/null 2>&1 && exit 1
  [ ! -e "$WORK/bin_f2/typeburn" ] || exit 1
); then ok "(f) explicit VERSION=-rc.test refused"
else bad "(f) VERSION rc override"; fi

# === (g) malicious archive member (symlink) → reject ====================
bd="$WORK/bin_g"; GDIR="$WORK/dl_g/$STABLE_TAG"; gstage="$WORK/gstage"
mkdir -p "$GDIR" "$gstage"
ln -s /etc/passwd "$gstage/typeburn"
tar -czf "$GDIR/typeburn_${STABLE_VER}_linux_amd64.tar.gz" -C "$gstage" typeburn
printf '%s  %s\n' \
  "$(sha256_of "$GDIR/typeburn_${STABLE_VER}_linux_amd64.tar.gz")" \
  "typeburn_${STABLE_VER}_linux_amd64.tar.gz" >"$GDIR/checksums.txt"
if (
  export BIN_DIR="$bd" TYPEBURN_UNAME_S=Linux TYPEBURN_UNAME_M=x86_64 \
         TYPEBURN_BASE_URL="$BASE/dl_g"
  sh "$INSTALL_SH" >/dev/null 2>&1 && exit 1
  if [ -e "$bd/typeburn" ] || [ -L "$bd/typeburn" ]; then exit 1; fi
); then ok "(g) symlink archive member rejected"
else bad "(g) symlink member"; fi

# === (h) failed install leaves prior binary byte-identical ==============
bd="$WORK/bin_h"; mkdir -p "$bd"
printf 'OLD-BINARY-SENTINEL\n' >"$bd/typeburn"; chmod 755 "$bd/typeburn"
before="$(sha256_of "$bd/typeburn")"
HDIR="$WORK/dl_h/$STABLE_TAG"; mkdir -p "$HDIR"
cp "$DOCROOT/dl/$STABLE_TAG"/*.tar.gz "$HDIR/"
sed 's/^[0-9a-f]\{64\}/deadbeef00000000000000000000000000000000000000000000000000000000/' \
  "$DOCROOT/dl/$STABLE_TAG/checksums.txt" >"$HDIR/checksums.txt"
if (
  export BIN_DIR="$bd" TYPEBURN_UNAME_S=Linux TYPEBURN_UNAME_M=x86_64 \
         TYPEBURN_BASE_URL="$BASE/dl_h"
  sh "$INSTALL_SH" >/dev/null 2>&1 && exit 1
  after="$(sha256_of "$bd/typeburn")"
  [ "$before" = "$after" ] || exit 1
); then ok "(h) failed install left prior binary byte-identical"
else bad "(h) prior binary integrity"; fi

# --- Report ---------------------------------------------------------------
echo "----------------------------------------"
echo "install.sh harness: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
