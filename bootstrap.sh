#!/usr/bin/env bash

set -e

echo "🔗 Deploying dotfiles with GNU Stow..."

# Verify stow is installed
if ! command -v stow &> /dev/null; then
    echo "❌ Error: GNU Stow is not installed"
    echo "Run ./install.sh first to install required tools"
    exit 1
fi

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

# Check for existing configs and handle conflicts
CONFIGS_TO_CHECK=(
    "$HOME/.zshrc"
    "$HOME/.gitconfig"
    "$HOME/.gitignore_global"
    "$HOME/.vimrc"
    "$HOME/.config/ghostty/config"
    "$HOME/.config/k9s/config.yaml"
)

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

# Check each config for conflicts
for config in "${CONFIGS_TO_CHECK[@]}"; do
    if [ -e "$config" ] && [ ! -L "$config" ]; then
        handle_conflict "$config"
    fi
done

# Prompt for git personalization
prompt_git_config

# Get the directory where this script is located
DOTFILES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DOTFILES_DIR"

# List of tool directories to stow
TOOLS=(
    zsh
    git
    vim
    ghostty
    k9s
)

echo ""
echo "🔗 Creating symlinks..."

for tool in "${TOOLS[@]}"; do
    if [ -d "$tool" ]; then
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
echo "  3. Delete backup if no longer needed: rm -rf $BACKUP_DIR"
