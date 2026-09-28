#!/usr/bin/env bash
# Isolated host-side NeferWL smoke test; no checkout or sibling module is modified.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
if [[ -z "${LIBWLDEVICES_HEADLESS:-}" || ! -x "$LIBWLDEVICES_HEADLESS" ]]; then
  echo 'LIBWLDEVICES_HEADLESS must point to an executable NeferWL binary' >&2
  exit 1
fi
TMP="$(mktemp -d)"
PID=""
cleanup() {
  result=$?
  trap - EXIT INT TERM
  if [[ -n "$PID" ]]; then
    kill -- -"$PID" 2>/dev/null || true
    # Bound cleanup if the compositor ignores TERM; never leave its process group.
    for ((j=0;j<20;j++)); do
      if ! kill -0 "$PID" 2>/dev/null; then break; fi
      sleep .05
    done
    if kill -0 "$PID" 2>/dev/null; then kill -KILL -- -"$PID" 2>/dev/null || true; fi
    wait "$PID" 2>/dev/null || true
  fi
  if [[ $result -ne 0 && -f "$TMP/neferwl.log" ]]; then cat "$TMP/neferwl.log" >&2; fi
  rm -rf -- "$TMP"
  exit "$result"
}
trap cleanup EXIT INT TERM
cp "$ROOT"/test/consumer/{go.mod,go.sum,fixture.go,main.go} "$TMP"/
(
  cd "$TMP"
  GOWORK=off go mod edit -replace="github.com/bnema/libwldevices-go=$ROOT"
  GOWORK=off go mod tidy
  CGO_ENABLED=0 GOWORK=off go build -o consumer .
)
mkdir -m 700 "$TMP/runtime" "$TMP/config"
# No host display or Wayland socket leaks into this session.
env -u DISPLAY -u WAYLAND_DISPLAY XDG_RUNTIME_DIR="$TMP/runtime" XDG_CONFIG_HOME="$TMP/config" \
  setsid "$LIBWLDEVICES_HEADLESS" --backend=headless --no-terminal --no-xwayland \
  --size 640x480 --timeout 20s >"$TMP/neferwl.log" 2>&1 &
PID=$!
# NeferWL uses wayland-1 unless another name is allocated. Bounded readiness.
for ((i=0;i<100;i++)); do
  if ! kill -0 "$PID" 2>/dev/null; then echo 'compositor exited before socket readiness' >&2; exit 1; fi
  for socket in "$TMP"/runtime/wayland-*; do
    if [[ -S "$socket" ]]; then
      env -u DISPLAY XDG_RUNTIME_DIR="$TMP/runtime" WAYLAND_DISPLAY="$(basename "$socket")" \
        timeout 15s "$TMP/consumer"
      exit
    fi
  done
  sleep .1
done
echo 'compositor socket readiness timed out' >&2
exit 1
