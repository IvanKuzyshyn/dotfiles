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

# ── Selection menu ──────────────────────────────────────────────────

# Map tool names to human-readable descriptions
get_tool_desc() {
    case "$1" in
        zsh)     echo "shell configuration" ;;
        git)     echo "git settings and aliases" ;;
        vim)     echo "editor configuration" ;;
        ghostty) echo "terminal emulator" ;;
        k9s)     echo "Kubernetes UI" ;;
        claude)  echo "AI assistant settings" ;;
        mise)    echo "dev tools and env manager" ;;
        *)       echo "" ;;
    esac
}

# Auto-detect configs from configs/ directory
for dir in "$SCRIPT_DIR"/configs/*/; do
    tool=$(basename "$dir")
    add_item "$tool" "$(get_tool_desc "$tool")"
done

show_selection_menu "Select configs to deploy (all selected by default):"

# ── Conflict checking ───────────────────────────────────────────────

# Return config file paths for a given tool (used for conflict detection)
get_config_paths() {
    case "$1" in
        zsh)     echo "$HOME/.zshrc" ;;
        git)     echo "$HOME/.gitconfig $HOME/.gitignore_global" ;;
        vim)     echo "$HOME/.vimrc" ;;
        ghostty) echo "$HOME/.config/ghostty/config" ;;
        k9s)     echo "$HOME/.config/k9s/config.yaml" ;;
        claude)  echo "$HOME/.claude/settings.json" ;;
        mise)    echo "$HOME/.config/mise/config.toml" ;;
    esac
}

# Function to prompt for git configuration
prompt_git_config() {
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

# Create backup directory with timestamp
BACKUP_DIR="$HOME/.dotfiles_backup_$(date +%Y%m%d_%H%M%S)"
BACKUP_CREATED=false

# Function to handle a single conflicting file
handle_conflict() {
    local config="$1"
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

# Check for conflicts only for selected configs
for tool in "${MENU_ITEMS[@]}"; do
    is_selected "$tool" || continue
    for config in $(get_config_paths "$tool"); do
        if [ -e "$config" ] && [ ! -L "$config" ]; then
            handle_conflict "$config"
        fi
    done
done

# Prompt for git personalization only if git config is selected
if is_selected "git"; then
    prompt_git_config
fi

cd "$SCRIPT_DIR"

echo ""
echo "🔗 Creating symlinks..."

for tool in "${MENU_ITEMS[@]}"; do
    is_selected "$tool" || continue
    if [ -d "configs/$tool" ]; then
        echo "  📁 Stowing $tool..."
        stow -v "$tool"
    else
        echo "  ⚠️  Skipping $tool (directory not found)"
    fi
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
