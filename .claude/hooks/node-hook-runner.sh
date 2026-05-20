#!/usr/bin/env bash
set -u

# Runs ClaudeKit Node hooks without turning a missing Node.js runtime into a
# startup crash loop. Native Claude Code installs do not bundle `node`, so
# Windows users can otherwise see one "node: command not found" error per hook.

HOOK_SCRIPT="${1:-}"
if [ -z "$HOOK_SCRIPT" ]; then
  exit 0
fi
shift || true

run_with_node() {
  local node_bin="$1"
  shift || true
  if [ -z "$node_bin" ]; then
    return 1
  fi
  exec "$node_bin" "$HOOK_SCRIPT" "$@"
}

if command -v node >/dev/null 2>&1; then
  run_with_node "$(command -v node)" "$@"
fi

if command -v node.exe >/dev/null 2>&1; then
  run_with_node "$(command -v node.exe)" "$@"
fi

to_posix_path() {
  local candidate="$1"
  if command -v cygpath >/dev/null 2>&1; then
    cygpath -u "$candidate" 2>/dev/null || printf '%s\n' "$candidate"
  else
    printf '%s\n' "$candidate"
  fi
}

try_windows_node_path() {
  local candidate="$1"
  shift || true
  if [ -z "$candidate" ]; then
    return 1
  fi

  local posix_candidate
  posix_candidate="$(to_posix_path "$candidate")"
  if [ -x "$posix_candidate" ]; then
    run_with_node "$posix_candidate" "$@"
  fi

  if [ -x "$candidate" ]; then
    run_with_node "$candidate" "$@"
  fi
}

PROGRAMFILES_X86="$(printenv 'ProgramFiles(x86)' 2>/dev/null || true)"

try_windows_node_path "${ProgramFiles:-}/nodejs/node.exe" "$@"
try_windows_node_path "${PROGRAMFILES:-}/nodejs/node.exe" "$@"
try_windows_node_path "${PROGRAMFILES_X86:-}/nodejs/node.exe" "$@"
try_windows_node_path "${NVM_SYMLINK:-}/node.exe" "$@"
try_windows_node_path "${NVM_HOME:-}/current/node.exe" "$@"
try_windows_node_path "${LOCALAPPDATA:-}/Programs/nodejs/node.exe" "$@"

case "$HOOK_SCRIPT" in
  *statusline.cjs)
    exit 0
    ;;
esac

warning_dir="${TMPDIR:-${TEMP:-/tmp}}"
warning_marker="$warning_dir/claudekit-node-hook-runner.warned"
if [ ! -e "$warning_marker" ]; then
  {
    printf '%s\n' "[ClaudeKit] Node.js 18+ was not found on PATH, so ClaudeKit hooks were skipped."
    printf '%s\n' "[ClaudeKit] Install Node.js from https://nodejs.org/ and restart Claude Code."
  } >&2
  : >"$warning_marker" 2>/dev/null || true
fi

exit 0
