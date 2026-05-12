#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/menu.sh"
parse_flags "$@"

echo "🔗 Deploying dotfiles with GNU Stow..."

# Verify stow is installed
if ! command -v stow &> /dev/null; then
    echo "❌ Error: GNU Stow is not installed"
    echo "Run ./install.sh first to install required tools"
    exit 1
fi

# ── Helpers ────────────────────────────────────────────────────────

# Map tool names to human-readable descriptions
get_tool_desc() {
    case "$1" in
        zsh)     echo "shell configuration" ;;
        git)     echo "git settings and aliases" ;;
        vim)     echo "editor configuration" ;;
        ghostty) echo "terminal emulator" ;;
        k9s)     echo "Kubernetes UI" ;;
        claude)  echo "Claude settings" ;;
        mise)    echo "dev tools and env manager" ;;
        *)       echo "" ;;
    esac
}

# Detect conflicts by running stow in dry-run mode.
# Populates the CONFLICTS array with absolute paths.
detect_conflicts() {
    local tool="$1"
    CONFLICTS=()
    local output
    output=$(stow -n -v "$tool" 2>&1) || true
    while IFS= read -r line; do
        # Pattern 1: "existing target is not owned by stow: <path>"
        if [[ "$line" =~ "existing target is not owned by stow: "(.*) ]]; then
            CONFLICTS+=("$HOME/${BASH_REMATCH[1]}")
        # Pattern 2: "cannot stow ... over existing target <path> since neither a link nor a directory"
        elif [[ "$line" =~ "over existing target "(.*)" since" ]]; then
            CONFLICTS+=("$HOME/${BASH_REMATCH[1]}")
        fi
    done <<< "$output"
}

# Function to prompt for git configuration
prompt_git_config() {
    if [ "$NON_INTERACTIVE" = true ]; then
        if [ -f "$HOME/.gitconfig.local" ]; then
            echo "✅ Keeping existing git configuration (~/.gitconfig.local)"
            return 0
        else
            echo "⚠️  ~/.gitconfig.local not found — run bootstrap.sh interactively to configure"
            return 0
        fi
    fi

    echo ""
    echo "⚙️  Git Configuration"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    # Check if .gitconfig.local already exists
    if [ -f "$HOME/.gitconfig.local" ]; then
        echo "📝 Existing git configuration found:"
        git config -f "$HOME/.gitconfig.local" user.name 2>/dev/null && \
            echo "   Name:  $(git config -f "$HOME/.gitconfig.local" user.name)"
        git config -f "$HOME/.gitconfig.local" user.email 2>/dev/null && \
            echo "   Email: $(git config -f "$HOME/.gitconfig.local" user.email)"
        echo ""
        read -p "Do you want to update your git configuration? (y/N): " update_config
        if [[ ! "$update_config" =~ ^[Yy]$ ]]; then
            echo "✅ Keeping existing git configuration"
            return 0
        fi
        echo ""
    fi

    # Prompt for name
    read -p "Enter your git name: " git_name
    while [[ -z "$git_name" ]]; do
        echo "❌ Name cannot be empty"
        read -p "Enter your git name: " git_name
    done

    # Prompt for email
    read -p "Enter git email address: " git_email
    while [[ -z "$git_email" ]]; do
        echo "❌ Email cannot be empty"
        read -p "Enter git email address: " git_email
    done

    # Create .gitconfig.local
    cat > "$HOME/.gitconfig.local" <<EOF
[user]
    name = $git_name
    email = $git_email
EOF

    echo ""
    echo "✅ Git configuration saved to ~/.gitconfig.local"
    echo "   Name:  $git_name"
    echo "   Email: $git_email"
}

# Create backup directory with timestamp (shared across all tools)
BACKUP_DIR="$HOME/.dotfiles_backup_$(date +%Y%m%d_%H%M%S)"
BACKUP_CREATED=false

# Function to handle a single conflicting file
handle_conflict() {
    local config="$1"

    if [ "$AUTO_BACKUP" = true ]; then
        if [ "$BACKUP_CREATED" = false ]; then
            mkdir -p "$BACKUP_DIR"
            BACKUP_CREATED=true
            echo "📦 Backup directory: $BACKUP_DIR"
        fi
        mkdir -p "$BACKUP_DIR/$(dirname "$config")"
        cp -r "$config" "$BACKUP_DIR/$config"
        rm -rf "$config"
        echo "   ✅ Auto-backed up and removed: $config"
        return
    fi

    echo ""
    echo "⚠️  Conflict: $config already exists"
    echo "   [b] Back up and remove"
    echo "   [r] Remove without backup"
    echo "   [s] Skip (stow will fail for this tool)"
    read -p "   Choose action (b/r/s): " action

    case "$action" in
        [Bb])
            if [ "$BACKUP_CREATED" = false ]; then
                mkdir -p "$BACKUP_DIR"
                BACKUP_CREATED=true
                echo "📦 Backup directory: $BACKUP_DIR"
            fi
            mkdir -p "$BACKUP_DIR/$(dirname "$config")"
            cp -r "$config" "$BACKUP_DIR/$config"
            rm -rf "$config"
            echo "   ✅ Backed up and removed: $config"
            ;;
        [Rr])
            rm -rf "$config"
            echo "   ✅ Removed: $config"
            ;;
        [Ss]|*)
            echo "   ⏭️ Skipped: $config"
            ;;
    esac
}

# ── Item registry ──────────────────────────────────────────────────

# Auto-detect configs from configs/ directory
for dir in "$SCRIPT_DIR"/configs/*/; do
    tool=$(basename "$dir")
    add_item "$tool" "$(get_tool_desc "$tool")"
done

# ── Per-tool deploy loop ───────────────────────────────────────────

cd "$SCRIPT_DIR"

echo ""

for tool in "${MENU_ITEMS[@]}"; do
    if ! confirm_item "$tool" "Deploy"; then
        continue
    fi

    # Detect and handle conflicts via stow dry run
    detect_conflicts "$tool"
    for config in "${CONFLICTS[@]}"; do
        handle_conflict "$config"
    done

    # Git personalization
    if [ "$tool" = "git" ]; then
        prompt_git_config
    fi

    # Stow
    if [ -d "configs/$tool" ]; then
        echo "  📁 Stowing $tool..."
        stow -v "$tool"
    else
        echo "  ⚠️  Skipping $tool (directory not found)"
    fi

    # Process .sample files: copy or merge into their final names
    # e.g. foo.sample.json → foo.json, bar.sample.md → bar.md
    while IFS= read -r sample_src; do
        rel="${sample_src#configs/$tool/}"
        sample_dest="$HOME/$rel"
        target_dest="${sample_dest/.sample/}"
        if [ ! -f "$target_dest" ]; then
            cp "$sample_dest" "$target_dest"
            echo "  📝 Created $(basename "$target_dest") from template"
        elif [[ "$target_dest" == *.json ]] && command -v jq &>/dev/null; then
            # Deep-merge JSON: sample as base, existing user file wins on conflicts,
            # arrays are combined with duplicates removed
            merged=$(jq -s '
                def deep_merge:
                    . as [$a, $b] |
                    if ($a | type) == "object" and ($b | type) == "object" then
                        ($a | keys_unsorted) + ($b | keys_unsorted) | unique |
                        map(. as $k |
                            if ($a | has($k)) and ($b | has($k)) then
                                {($k): ([$a[$k], $b[$k]] | deep_merge)}
                            elif ($b | has($k)) then {($k): $b[$k]}
                            else {($k): $a[$k]}
                            end
                        ) | add // {}
                    elif ($a | type) == "array" and ($b | type) == "array" then
                        ($a + $b) | unique
                    elif $b != null then $b
                    else $a
                    end;
                deep_merge
            ' "$sample_dest" "$target_dest" 2>/dev/null)

            if [ $? -eq 0 ] && [ -n "$merged" ]; then
                echo "$merged" > "$target_dest"
                echo "  🔀 Merged $(basename "$target_dest") (your settings take precedence)"
            else
                echo "  ⚠️  Could not merge $(basename "$target_dest"), keeping existing version"
            fi
        else
            echo "  ✅ $(basename "$target_dest") already exists, keeping current version"
        fi
    done < <(find "configs/$tool" -name "*.sample.*" 2>/dev/null || true)
done

echo ""
echo "✨ Dotfiles deployment complete!"

if [ "$BACKUP_CREATED" = true ]; then
    echo ""
    echo "📦 Backup location: $BACKUP_DIR"
fi

echo ""
echo "Next steps:"
echo "  1. Restart your shell: exec zsh"
echo "  2. Verify configs are working correctly"
if [ "$BACKUP_CREATED" = true ]; then
    echo "  3. Delete backup if no longer needed: rm -rf $BACKUP_DIR"
fi
