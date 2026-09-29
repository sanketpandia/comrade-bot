#!/usr/bin/env bash
#
# start-dev.sh — Local dev: backing services via Compose; Politburo and
# comrade-bot-discord on the host for live reload.
#
# Starts a tmux session "infinite-stage" with:
#  - window 1 (compose-up): compose up (db, redis, observability, swagger-editor, …)
#  - window 2 (comrade-bot): npm run dev
#  - window 3 (politburo): Air (optional; prefer VS Code debug instead)
#
# Uses Docker by default. For Podman: CONTAINER_CLI=podman ./start-dev.sh
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel 2>/dev/null || cd "$SCRIPT_DIR/../.." && pwd)"

SESSION="infinite-stage"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.dev.yml"
COMPOSE_CMD="${CONTAINER_CLI:-docker} compose"

tmux new-session -d -s "$SESSION" -n "compose-up"
tmux send-keys -t "$SESSION":1 "cd \"${SCRIPT_DIR}\" && $COMPOSE_CMD -f docker-compose.dev.yml up" C-m

tmux new-window -t "$SESSION":2 -n "comrade-bot"
tmux send-keys -t "$SESSION":2 "cd \"${REPO_ROOT}\" && make generate && cd services/comrade-bot-discord && npm run dev" C-m

tmux new-window -t "$SESSION":3 -n "politburo"
tmux send-keys -t "$SESSION":3 "cd \"${REPO_ROOT}\" && make generate && cd services/politburo && sh -c 'cd ../../cicd/tools && go tool air -c ../../services/politburo/.air.toml' 2>&1 | tee /tmp/politburo.log" C-m

tmux select-window -t "$SESSION":1
tmux attach-session -t "$SESSION"
