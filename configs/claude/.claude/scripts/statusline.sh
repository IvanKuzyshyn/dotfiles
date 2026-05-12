#!/usr/bin/env bash

# Custom statusline for Claude Code
# Shows: directory, branch, model, lines changed, duration, cost, context bar

input=$(cat)

# Extract all fields in a single jq call
{
	read -r model
	read -r current_dir
	read -r pct
	read -r added
	read -r removed
	read -r duration_ms
	read -r cost
} < <(echo "$input" | jq -r '
	.model.display_name,
	.workspace.current_dir,
	(.context_window.used_percentage // 0 | floor),
	(.cost.total_lines_added // 0),
	(.cost.total_lines_removed // 0),
	(.cost.total_duration_ms // 0),
	(.cost.total_cost_usd // 0)
')

RED='\033[0;31m'
GREEN='\033[0;32m'
GRAY='\033[0;90m'
NC='\033[0m'

# Context progress bar using printf -v (no loops)
BAR_WIDTH=15
FILLED=$((pct * BAR_WIDTH / 100))
EMPTY=$((BAR_WIDTH - FILLED))
BAR=""
[ "$FILLED" -gt 0 ] && printf -v FILL_STR "%${FILLED}s" && BAR="${FILL_STR// /█}"
[ "$EMPTY" -gt 0 ] && printf -v EMPTY_STR "%${EMPTY}s" && BAR="${BAR}${EMPTY_STR// /░}"

# Format duration
if [ "$duration_ms" -ge 3600000 ]; then
	DURATION="$((duration_ms / 3600000))h"
elif [ "$duration_ms" -ge 60000 ]; then
	DURATION="$((duration_ms / 60000))m"
elif [ "$duration_ms" -ge 1000 ]; then
	DURATION="$((duration_ms / 1000))s"
else
	DURATION="${duration_ms}ms"
fi

# Git branch
BRANCH=""
if git -C "$current_dir" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
	BRANCH=$(git -C "$current_dir" branch --show-current 2>/dev/null || echo "detached")
fi

# Build output
OUTPUT="/${current_dir##*/}"
[ -n "$BRANCH" ] && OUTPUT+=" ($BRANCH)"
OUTPUT+=" ${GRAY}|${NC} $model"
[ "$added" -gt 0 ] && OUTPUT+=" ${GREEN}+${added}${NC}"
[ "$removed" -gt 0 ] && OUTPUT+=" ${RED}-${removed}${NC}"
OUTPUT+=" in $DURATION"

# Format cost only if > 0
if awk -v c="$cost" 'BEGIN { exit !(c > 0) }'; then
	OUTPUT+=" for $(printf '$%.2f' "$cost")"
fi

OUTPUT+=" ${GRAY}|${NC} $BAR ${pct}%"

printf '%b\n' "$OUTPUT"
