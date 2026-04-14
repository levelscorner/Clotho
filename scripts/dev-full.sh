#!/usr/bin/env bash
# scripts/dev-full.sh — start the full Clotho stack for local development.
#
# Brings up every service needed to dogfood Clotho with local-only media
# generation: Postgres, Kokoro-FastAPI (TTS), ComfyUI (image gen),
# Clotho backend (with NO_AUTH bypass), Clotho frontend (Vite dev server).
#
# Services that are already listening on their expected port are left alone.
# Services this script starts are tracked in ~/.clotho-dev.pids so the
# paired `scripts/dev-stop.sh` can tear them down cleanly.
#
# Usage:
#   scripts/dev-full.sh          # start everything
#   scripts/dev-full.sh status   # show what's up
#   scripts/dev-full.sh logs     # tail all service logs
#
# Per-service behaviour is defined in tiny functions below — grep for
# `start_<name>` to find or tweak one.

set -euo pipefail

# --- Paths (edit here if your layout differs) -------------------------------

CLOTHO_ROOT="${CLOTHO_ROOT:-/Users/level/ws/projects/Clotho}"
MODELS_ROOT="${MODELS_ROOT:-/Users/level/ws/models}"
KOKORO_DIR="${KOKORO_DIR:-$MODELS_ROOT/Kokoro-FastAPI}"
COMFYUI_DIR="${COMFYUI_DIR:-$MODELS_ROOT/ComfyUI}"

PID_FILE="${HOME}/.clotho-dev.pids"
LOG_DIR="${HOME}/.clotho-dev-logs"
mkdir -p "$LOG_DIR"

# --- Helpers ---------------------------------------------------------------

c_reset=$'\033[0m'
c_green=$'\033[32m'
c_yellow=$'\033[33m'
c_red=$'\033[31m'
c_blue=$'\033[34m'
c_dim=$'\033[2m'

say()  { printf "%s[clotho]%s %s\n" "$c_blue" "$c_reset" "$*"; }
ok()   { printf "  %s✓%s %s\n" "$c_green" "$c_reset" "$*"; }
warn() { printf "  %s!%s %s\n" "$c_yellow" "$c_reset" "$*"; }
err()  { printf "  %s✗%s %s\n" "$c_red" "$c_reset" "$*"; }

is_listening() {
  lsof -i ":$1" -sTCP:LISTEN -t >/dev/null 2>&1
}

record_pid() {
  # record_pid <name> <pid>
  echo "$1 $2" >> "$PID_FILE"
}

# --- Service: Postgres (via docker compose) --------------------------------

start_postgres() {
  if is_listening 5432; then
    ok "postgres already listening on :5432"
    return
  fi
  if ! docker version >/dev/null 2>&1; then
    err "docker daemon not running — start Docker Desktop, then re-run"
    return 1
  fi
  say "starting postgres..."
  ( cd "$CLOTHO_ROOT" && docker compose up -d postgres ) > "$LOG_DIR/postgres.log" 2>&1
  # Wait up to 15s for Postgres to accept connections
  for i in {1..15}; do
    if is_listening 5432; then ok "postgres up (took ${i}s)"; return; fi
    sleep 1
  done
  err "postgres did not bind :5432 within 15s; see $LOG_DIR/postgres.log"
  return 1
}

# --- Service: Kokoro-FastAPI (local TTS, :8880) ----------------------------

start_kokoro() {
  if is_listening 8880; then
    ok "kokoro already listening on :8880"
    return
  fi
  if [ ! -d "$KOKORO_DIR/.venv" ]; then
    warn "kokoro venv missing at $KOKORO_DIR/.venv — skipping (install via setup)"
    return
  fi
  say "starting kokoro..."
  (
    cd "$KOKORO_DIR"
    export USE_GPU=true
    export USE_ONNX=false
    export DEVICE_TYPE=mps
    export PYTORCH_ENABLE_MPS_FALLBACK=1
    export PYTHONPATH="$KOKORO_DIR:$KOKORO_DIR/api"
    export MODEL_DIR=src/models
    export VOICES_DIR=src/voices/v1_0
    nohup .venv/bin/uvicorn api.src.main:app \
      --host 127.0.0.1 --port 8880 \
      > "$LOG_DIR/kokoro.log" 2>&1 &
    echo $! > /tmp/kokoro.pid
    disown $! 2>/dev/null || true
  )
  record_pid kokoro "$(cat /tmp/kokoro.pid)"
  # Kokoro takes 10-30s to load the voice model on first start.
  for i in {1..40}; do
    if is_listening 8880; then ok "kokoro up (took ${i}s)"; return; fi
    sleep 1
  done
  err "kokoro did not bind :8880 within 40s; tail $LOG_DIR/kokoro.log"
}

# --- Service: ComfyUI (local image gen, :8188) -----------------------------

start_comfyui() {
  if is_listening 8188; then
    ok "comfyui already listening on :8188"
    return
  fi
  if [ ! -d "$COMFYUI_DIR/.venv" ]; then
    warn "comfyui venv missing at $COMFYUI_DIR/.venv — skipping"
    return
  fi
  if [ ! -f "$COMFYUI_DIR/models/unet/flux1-schnell-fp8.safetensors" ] && \
     [ ! -f "$COMFYUI_DIR/models/checkpoints/flux1-schnell-fp8.safetensors" ]; then
    warn "FLUX checkpoint missing — image gen will 500 until it downloads"
  fi
  say "starting comfyui..."
  (
    cd "$COMFYUI_DIR"
    nohup .venv/bin/python main.py \
      --listen 127.0.0.1 --port 8188 --disable-auto-launch \
      > "$LOG_DIR/comfyui.log" 2>&1 &
    echo $! > /tmp/comfyui.pid
    disown $! 2>/dev/null || true
  )
  record_pid comfyui "$(cat /tmp/comfyui.pid)"
  for i in {1..60}; do
    if is_listening 8188; then ok "comfyui up (took ${i}s)"; return; fi
    sleep 1
  done
  err "comfyui did not bind :8188 within 60s; tail $LOG_DIR/comfyui.log"
}

# --- Service: Clotho backend (Go, :8080) -----------------------------------

start_backend() {
  if is_listening 8080; then
    ok "backend already listening on :8080"
    return
  fi
  say "starting backend (NO_AUTH=true)..."
  (
    cd "$CLOTHO_ROOT"
    export NO_AUTH=true
    export CLOTHO_ACKNOWLEDGE_NO_AUTH=yes
    export OLLAMA_URL="${OLLAMA_URL:-http://localhost:11434}"
    export KOKORO_URL="${KOKORO_URL:-http://localhost:8880}"
    export COMFYUI_URL="${COMFYUI_URL:-http://localhost:8188}"
    # REPLICATE_API_TOKEN passes through if the caller set it — used only for video.
    nohup go run ./cmd/clotho > "$LOG_DIR/backend.log" 2>&1 &
    echo $! > /tmp/clotho-backend.pid
    disown $! 2>/dev/null || true
  )
  record_pid backend "$(cat /tmp/clotho-backend.pid)"
  for i in {1..30}; do
    if is_listening 8080; then ok "backend up (took ${i}s)"; return; fi
    sleep 1
  done
  err "backend did not bind :8080 within 30s; tail $LOG_DIR/backend.log"
}

# --- Service: Clotho frontend (Vite, :3000) --------------------------------

start_frontend() {
  if is_listening 3000; then
    ok "frontend already listening on :3000"
    return
  fi
  say "starting frontend (VITE_NO_AUTH=true)..."
  (
    cd "$CLOTHO_ROOT/web"
    export VITE_NO_AUTH=true
    nohup npm run dev > "$LOG_DIR/frontend.log" 2>&1 &
    echo $! > /tmp/clotho-frontend.pid
    disown $! 2>/dev/null || true
  )
  record_pid frontend "$(cat /tmp/clotho-frontend.pid)"
  for i in {1..30}; do
    if is_listening 3000; then ok "frontend up (took ${i}s)"; return; fi
    sleep 1
  done
  err "frontend did not bind :3000 within 30s; tail $LOG_DIR/frontend.log"
}

# --- Commands --------------------------------------------------------------

cmd_up() {
  : > "$PID_FILE"   # clear prior run's record
  start_postgres
  start_kokoro
  start_comfyui
  start_backend
  start_frontend
  echo ""
  printf "%s=== Clotho local stack ===%s\n" "$c_blue" "$c_reset"
  printf "  app       http://localhost:3000\n"
  printf "  api       http://localhost:8080/health\n"
  printf "  kokoro    http://localhost:8880/web/ (FastAPI docs)\n"
  printf "  comfyui   http://localhost:8188\n"
  printf "  postgres  localhost:5432\n"
  printf "  %slogs%s      tail -f ~/.clotho-dev-logs/*.log\n" "$c_dim" "$c_reset"
  printf "  %sstop%s      make dev-stop\n" "$c_dim" "$c_reset"
}

cmd_status() {
  printf "%-10s %-6s %s\n" SERVICE PORT STATUS
  for svc in "postgres:5432" "kokoro:8880" "comfyui:8188" "backend:8080" "frontend:3000"; do
    name="${svc%:*}"
    port="${svc#*:}"
    if is_listening "$port"; then
      printf "%-10s %-6s %sup%s\n" "$name" "$port" "$c_green" "$c_reset"
    else
      printf "%-10s %-6s %sdown%s\n" "$name" "$port" "$c_red" "$c_reset"
    fi
  done
}

cmd_logs() {
  if command -v multitail >/dev/null; then
    multitail "$LOG_DIR"/*.log
  else
    tail -f "$LOG_DIR"/*.log
  fi
}

case "${1:-up}" in
  up)     cmd_up ;;
  status) cmd_status ;;
  logs)   cmd_logs ;;
  *)      echo "usage: $0 [up|status|logs]" ; exit 1 ;;
esac
