#!/usr/bin/env bash
# Editor contract tests. Composition cases open a GPUI test window, so they
# run under Xvfb when a display is missing. Exit status is cargo's.
set -u
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
export CARGO_BUILD_JOBS="${CARGO_BUILD_JOBS:-2}"
export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/tmp/runtime-$(id -u)}"
mkdir -p "$XDG_RUNTIME_DIR"
chmod 700 "$XDG_RUNTIME_DIR"
if [[ -z "${DISPLAY:-}" ]]; then
  exec xvfb-run -a cargo test --manifest-path "$root/Cargo.toml" --no-fail-fast -- --test-threads=1
fi
exec cargo test --manifest-path "$root/Cargo.toml" --no-fail-fast -- --test-threads=1
