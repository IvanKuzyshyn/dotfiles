#!/usr/bin/env bash
#
# Shared confirmation prompt library for dotfiles scripts.
# Source this file, then use add_item/confirm_item.
#
# Flags:
#   --all      Skip prompts — confirm everything automatically
#   --backup   Auto-backup conflicts without prompting (bootstrap.sh)

# ── State ─────────────────────────────────────────────────────────

NON_INTERACTIVE=false
AUTO_BACKUP=false

MENU_ITEMS=()
MENU_DESCS=()

# ── Helpers ───────────────────────────────────────────────────────

parse_flags() {
    for arg in "$@"; do
        case "$arg" in
            --all)    NON_INTERACTIVE=true ;;
            --backup) AUTO_BACKUP=true ;;
        esac
    done
}

add_item() {
    MENU_ITEMS+=("$1")
    MENU_DESCS+=("${2:-}")
}

# Prompt the user to confirm a single item.
# Returns 0 (yes) or 1 (skip).
#
# Usage: confirm_item "homebrew-tools" "Install"
#   verb defaults to "Install" if omitted.
confirm_item() {
    local name="$1"
    local verb="${2:-Install}"

    # Look up description from registry
    local desc=""
    for i in "${!MENU_ITEMS[@]}"; do
        if [ "${MENU_ITEMS[$i]}" = "$name" ]; then
            desc="${MENU_DESCS[$i]}"
            break
        fi
    done

    # Non-interactive mode: always yes
    if [ "$NON_INTERACTIVE" = true ]; then
        if [ -n "$desc" ]; then
            echo "  $verb $name ($desc)"
        else
            echo "  $verb $name"
        fi
        return 0
    fi

    # Build prompt string
    local prompt="$verb $name"
    [ -n "$desc" ] && prompt="$prompt ($desc)"
    prompt="$prompt? [Y/n] "

    read -r -p "$prompt" answer
    case "$answer" in
        [Nn]) return 1 ;;
        *)    return 0 ;;
    esac
}
