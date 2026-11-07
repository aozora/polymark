#!/usr/bin/env bash
# Open the GPUiX shell on a virtual X11 display and grab the root window.
# TestGpuixRenderer cannot read pixels back on Linux (upstream 0.11.0).
# This uses the real GpuixRenderer. A missing window or a failed grab is
# recorded and is not treated as a passed visual test.
set -u
root="$(cd "$(dirname "$0")/.." && pwd)"
node="${POLYMARK_NODE:-$HOME/.nvm/versions/node/v24.22.2/bin/node}"
out="$root/docs/evidence/window-shell.png"
log="$root/docs/evidence/window-shell.log"
mkdir -p "$root/docs/evidence" "${XDG_RUNTIME_DIR:-/tmp/runtime-$(id -u)}"
export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/tmp/runtime-$(id -u)}"
chmod 700 "$XDG_RUNTIME_DIR"
export VK_DRIVER_FILES="${VK_DRIVER_FILES:-/usr/share/vulkan/icd.d/lvp_icd.json}"
export PATH="$(dirname "$node"):$PATH"

{
  echo "date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "node=$($node -v)"
  echo "display_driver=$VK_DRIVER_FILES"
} >"$log"

xvfb-run -a -s "-screen 0 1280x800x24" bash -c '
  set -u
  echo "DISPLAY=$DISPLAY" >> "'"$log"'"
  "'"$node"'" "'"$root"'/node_modules/gpuix-svelte/bin/gpuix-svelte.js" "'"$root"'/app/app.ts" >> "'"$log"'" 2>&1 &
  app_pid=$!
  sleep 8
  if ! kill -0 "$app_pid" 2>/dev/null; then
    echo "app_exited_before_capture=1" >> "'"$log"'"
    exit 3
  fi
  if command -v import >/dev/null 2>&1; then
    import -window root -display "$DISPLAY" "'"$out"'" >> "'"$log"'" 2>&1
    cap=$?
  elif command -v xwd >/dev/null 2>&1 && command -v convert >/dev/null 2>&1; then
    xwd -root -silent | convert xwd:- "'"$out"'" >> "'"$log"'" 2>&1
    cap=$?
  else
    echo "no_screenshot_tool" >> "'"$log"'"
    cap=4
  fi
  kill "$app_pid" 2>/dev/null || true
  wait "$app_pid" 2>/dev/null || true
  echo "capture_exit=$cap" >> "'"$log"'"
  exit "$cap"
' >>"$log" 2>&1
status=$?
echo "script_exit=$status" >>"$log"
echo "wrote $log"
exit "$status"
