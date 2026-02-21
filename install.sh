#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/menu.sh"
parse_flags "$@"

echo "🚀 Starting tool installation..."

# Platform detection
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "❌ Error: This script currently only supports macOS"
    echo "Please install tools manually on your platform"
    exit 1
fi

echo "✅ Platform: macOS"

# ── Item registry ──────────────────────────────────────────────────

add_item "homebrew-tools" "15 packages via Brewfile"
add_item "rust"           "via rustup"
add_item "oh-my-zsh"      "zsh framework"
add_item "nvm"            "Node.js version manager"
add_item "claude-code"    "AI coding assistant"

# ── Installation ────────────────────────────────────────────────────

echo ""

# Install Homebrew packages and casks via Brewfile
if confirm_item "homebrew-tools"; then
    if ! command -v brew &> /dev/null; then
        echo "📦 Installing Homebrew..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    else
        echo "✅ Homebrew already installed"
    fi

    echo "🔄 Updating Homebrew..."
    brew update

    echo "📦 Installing Homebrew packages and casks via Brewfile..."
    brew bundle --file="$SCRIPT_DIR/Brewfile"
fi

# Install Rust via rustup
if confirm_item "rust"; then
    if command -v rustc &> /dev/null; then
        echo "✅ Rust already installed ($(rustc --version))"
    else
        echo "🦀 Installing Rust via rustup..."
        curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
        source "$HOME/.cargo/env"
        echo "✅ Rust installed ($(rustc --version))"
    fi
fi

# Install oh-my-zsh
if confirm_item "oh-my-zsh"; then
    if [ -d "$HOME/.oh-my-zsh" ]; then
        echo "✅ oh-my-zsh already installed"
    else
        echo "📦 Installing oh-my-zsh..."
        sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended
        echo "✅ oh-my-zsh installed"
    fi

    # Install zsh plugins
    ZSH_CUSTOM="${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}"

    if [ -d "$ZSH_CUSTOM/plugins/zsh-autosuggestions" ]; then
        echo "✅ zsh-autosuggestions already installed"
    else
        echo "📦 Installing zsh-autosuggestions..."
        git clone https://github.com/zsh-users/zsh-autosuggestions "$ZSH_CUSTOM/plugins/zsh-autosuggestions"
        echo "✅ zsh-autosuggestions installed"
    fi

    if [ -d "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting" ]; then
        echo "✅ zsh-syntax-highlighting already installed"
    else
        echo "📦 Installing zsh-syntax-highlighting..."
        git clone https://github.com/zsh-users/zsh-syntax-highlighting "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting"
        echo "✅ zsh-syntax-highlighting installed"
    fi
fi

# Install nvm
if confirm_item "nvm"; then
    if [ -d "$HOME/.nvm" ]; then
        echo "✅ nvm already installed"
    else
        echo "📦 Installing nvm..."
        curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
    fi

    # Load nvm for this session
    export NVM_DIR="$HOME/.nvm"
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

    # Install Node.js LTS via nvm if not present
    if command -v node &> /dev/null; then
        echo "✅ Node.js already installed ($(node --version))"
    else
        echo "📦 Installing Node.js LTS via nvm..."
        nvm install --lts
        nvm use --lts
        echo "✅ Node.js installed ($(node --version))"
    fi
fi

# Install Claude Code via npm
if confirm_item "claude-code"; then
    # Load nvm if available (may have been installed earlier or in a previous run)
    export NVM_DIR="$HOME/.nvm"
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

    if command -v claude &> /dev/null; then
        echo "✅ Claude Code already installed"
    elif command -v npm &> /dev/null; then
        echo "🤖 Installing Claude Code..."
        npm install -g @anthropic-ai/claude-code
        echo "✅ Claude Code installed"
    else
        echo "⚠️  Skipping Claude Code: npm not found (install nvm first to get Node.js)"
    fi
fi

echo ""
echo "✨ Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Run ./bootstrap.sh to deploy configurations"
echo "  2. Restart your shell or run: exec zsh"
