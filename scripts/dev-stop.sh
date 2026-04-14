#!/usr/bin/env bash
# scripts/dev-stop.sh — tear down the local Clotho stack started by dev-full.sh.
#
# Only stops services this machine started via dev-full.sh (tracked in
# ~/.clotho-dev.pids). Postgres via docker compose is stopped with
# `docker compose stop postgres` so your data volume survives.

set -euo pipefail

PID_FILE="${HOME}/.clotho-dev.pids"
CLOTHO_ROOT="${CLOTHO_ROOT:-/Users/level/ws/projects/Clotho}"

c_green=$'\033[32m'; c_yellow=$'\033[33m'; c_red=$'\033[31m'; c_reset=$'\033[0m'
ok()   { printf "  %s✓%s %s\n" "$c_green" "$c_reset" "$*"; }
warn() { printf "  %s!%s %s\n" "$c_yellow" "$c_reset" "$*"; }

kill_pid() {
  local name="$1" pid="$2"
  if [ -z "$pid" ] || ! kill -0 "$pid" 2>/dev/null; then
    warn "$name (pid $pid) not running"
    return
  fi
  kill "$pid" 2>/dev/null || true
  for _ in {1..10}; do
    if ! kill -0 "$pid" 2>/dev/null; then ok "$name (pid $pid) stopped"; return; fi
    sleep 0.3
  done
  kill -9 "$pid" 2>/dev/null || true
  warn "$name (pid $pid) force-killed"
}

if [ -f "$PID_FILE" ]; then
  while IFS=' ' read -r name pid; do
    [ -n "$name" ] && kill_pid "$name" "$pid"
  done < "$PID_FILE"
  rm -f "$PID_FILE"
else
  warn "no PID file at $PID_FILE — nothing tracked to stop"
fi

# Also drop any Vite / Go run child processes that detached from our PID file.
pkill -f "go run ./cmd/clotho" 2>/dev/null || true
pkill -f "vite" 2>/dev/null || true

# Stop Postgres last so active connections drain cleanly.
if docker version >/dev/null 2>&1; then
  ( cd "$CLOTHO_ROOT" && docker compose stop postgres >/dev/null 2>&1 ) && ok "postgres stopped (data preserved)"
fi
