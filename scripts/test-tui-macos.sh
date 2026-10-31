#!/usr/bin/env bash
# Scripted macOS interaction smoke test. It drives the TUI in tmux and checks
# text-level state changes; it cannot validate graphical image rendering.
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

if [ "$(uname -s)" != Darwin ]; then
  echo "warning: this script is intended for macOS" >&2
fi
command -v tmux >/dev/null 2>&1 || { echo "tmux is required" >&2; exit 1; }

make go-build >/dev/null

session="sparkstore-smoke-$$"
cleanup() {
  tmux kill-session -t "$session" 2>/dev/null || true
}
trap cleanup EXIT

tmux new-session -d -s "$session" -x 120 -y 40 './build/sparkstore'
capture() {
  tmux capture-pane -pt "$session" 2>/dev/null || true
}

failed=0
pass() { echo "PASS: $1"; }
fail() {
  echo "FAIL: $1"
  capture | head -30
  failed=1
}

sleep 7
if capture | grep -q "应用列表"; then pass "browse screen"; else fail "browse screen"; fi

before=$(capture | grep '›' | head -1 || true)
tmux send-keys -t "$session" j
sleep 1
after=$(capture | grep '›' | head -1 || true)
if [ "$before" != "$after" ]; then pass "arrow navigation"; else fail "arrow navigation"; fi

category_before=$(capture | head -1 || true)
tmux send-keys -t "$session" ']'
sleep 4
category_after=$(capture | head -1 || true)
if [ "$category_before" != "$category_after" ]; then pass "category switch"; else fail "category switch"; fi

if capture | grep -q "界面截图"; then pass "icon + screenshot pane"; else fail "icon + screenshot pane"; fi

tmux send-keys -t "$session" /
tmux send-keys -t "$session" -l Code
tmux send-keys -t "$session" Enter
sleep 1
if capture | grep -q "筛选："; then pass "search"; else fail "search"; fi

tmux send-keys -t "$session" q
sleep 1
if tmux has-session -t "$session" 2>/dev/null; then fail "quit"; else pass "quit"; fi

exit "$failed"
