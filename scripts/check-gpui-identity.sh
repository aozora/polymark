#!/usr/bin/env bash
# Compile the GPUI identity probe and keep the rustc diagnostic.
# A non-zero cargo status is the expected evidence. This script exits 0
# after writing the log so a caller can archive it. The log's cargo status
# is the last line `cargo_exit=N`.
set -u
root="$(cd "$(dirname "$0")/.." && pwd)"
log="$root/docs/evidence/gpui-identity-check.log"
mkdir -p "$root/docs/evidence"
export CARGO_BUILD_JOBS="${CARGO_BUILD_JOBS:-2}"
{
  echo "date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "rustc=$(rustc --version)"
  echo "probe=probes/gpui-identity gpuix rev 0ce4f41 gpui-pre 0.3.11 editor 0.2.0"
} >"$log"
set +e
cargo check --manifest-path "$root/probes/gpui-identity/Cargo.toml" >>"$log" 2>&1
status=$?
set -e
echo "cargo_exit=$status" >>"$log"
echo "wrote $log"
echo "cargo_exit=$status"
exit 0
