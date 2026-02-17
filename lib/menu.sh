#!/usr/bin/env bash
#
# Shared interactive selection menu for dotfiles scripts.
# Source this file, then use add_group/add_item/show_selection_menu/is_selected.
#
# Supports --all flag to skip the interactive menu (useful for CI).

# ── State ─────────────────────────────────────────────────────────

NON_INTERACTIVE=false

MENU_ITEMS=()
MENU_DESCS=()
MENU_SELECTED=()
GROUP_AT=()
GROUP_NAMES=()

# ── Helpers ───────────────────────────────────────────────────────

parse_flags() {
    for arg in "$@"; do
        case "$arg" in
            --all) NON_INTERACTIVE=true ;;
        esac
    done
}

add_group() {
    GROUP_AT+=("${#MENU_ITEMS[@]}")
    GROUP_NAMES+=("$1")
}

add_item() {
    MENU_ITEMS+=("$1")
    MENU_DESCS+=("${2:-}")
    MENU_SELECTED+=(1)
}

show_selection_menu() {
    local title="$1"

    if [ "$NON_INTERACTIVE" = true ]; then
        echo ""
        echo "$title"
        echo "(--all: selecting everything)"
        return
    fi

    while true; do
        echo ""
        echo "$title"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        local group_idx=0
        for i in "${!MENU_ITEMS[@]}"; do
            if [ "$group_idx" -lt "${#GROUP_AT[@]}" ] && [ "$i" -eq "${GROUP_AT[$group_idx]}" ]; then
                echo ""
                echo "  ${GROUP_NAMES[$group_idx]}:"
                group_idx=$((group_idx + 1))
            fi
            local check=" "
            [ "${MENU_SELECTED[$i]}" -eq 1 ] && check="x"
            local num=$((i + 1))
            if [ -n "${MENU_DESCS[$i]}" ]; then
                printf "  %2d) [%s] %-16s (%s)\n" "$num" "$check" "${MENU_ITEMS[$i]}" "${MENU_DESCS[$i]}"
            else
                printf "  %2d) [%s] %s\n" "$num" "$check" "${MENU_ITEMS[$i]}"
            fi
        done
        echo ""
        echo "  Toggle: type numbers (e.g. \"2 5\"), a = toggle all, Enter = proceed"
        read -r -p "  > " input
        [ -z "$input" ] && break
        if [ "$input" = "a" ] || [ "$input" = "A" ]; then
            local all=1
            for i in "${!MENU_SELECTED[@]}"; do
                [ "${MENU_SELECTED[$i]}" -eq 0 ] && all=0 && break
            done
            local val=1; [ "$all" -eq 1 ] && val=0
            for i in "${!MENU_SELECTED[@]}"; do MENU_SELECTED[$i]=$val; done
            continue
        fi
        for num in $input; do
            if [[ "$num" =~ ^[0-9]+$ ]]; then
                local idx=$((num - 1))
                if [ "$idx" -ge 0 ] && [ "$idx" -lt "${#MENU_ITEMS[@]}" ]; then
                    [ "${MENU_SELECTED[$idx]}" -eq 1 ] && MENU_SELECTED[$idx]=0 || MENU_SELECTED[$idx]=1
                fi
            fi
        done
    done
}

is_selected() {
    for i in "${!MENU_ITEMS[@]}"; do
        if [ "${MENU_ITEMS[$i]}" = "$1" ]; then
            [ "${MENU_SELECTED[$i]}" -eq 1 ] && return 0 || return 1
        fi
    done
    return 1
}
