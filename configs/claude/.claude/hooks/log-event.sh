#!/usr/bin/env bash
set -ue -o pipefail

# Universal hook logger - logs all hook events to JSONL

LOG_DIR="${CLAUDE_PROJECT_DIR:+${CLAUDE_PROJECT_DIR}/.claude}"
LOG_DIR="${LOG_DIR:-$HOME/.claude}"

# Validate path - no traversal
[[ "$LOG_DIR" == *".."* ]] && exit 0

mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/hook-events.jsonl"

EVENT_TYPE="${1:-unknown}"
input=$(cat)

# Build and append log entry in a single jq call
jq -n \
	--arg ts "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
	--arg event "$EVENT_TYPE" \
	--arg project_dir "${CLAUDE_PROJECT_DIR:-}" \
	--argjson input "$(
		if [ -z "$input" ]; then
			echo '{}'
		elif echo "$input" | jq empty 2>/dev/null; then
			echo "$input"
		else
			jq -n --arg raw "$input" '{raw_input: $raw}'
		fi
	)" \
	'{timestamp: $ts, event: $event, project_dir: $project_dir, input: $input}' \
	>>"$LOG_FILE"

exit 0
