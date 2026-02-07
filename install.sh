#!/usr/bin/env bash

set -e

echo "🚀 Starting tool installation..."

# Platform detection
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "❌ Error: This script currently only supports macOS"
    echo "Please install tools manually on your platform"
    exit 1
fi

echo "✅ Platform: macOS"

# Install Homebrew if not present
if ! command -v brew &> /dev/null; then
    echo "📦 Installing Homebrew..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
else
    echo "✅ Homebrew already installed"
fi

# Update Homebrew
echo "🔄 Updating Homebrew..."
brew update

# Install Homebrew packages
echo "📦 Installing Homebrew packages..."
PACKAGES=(
    stow
    k9s
    kubectl
    awscli
    gh
    jq
    yq
    ncdu
    git
    go
)

for package in "${PACKAGES[@]}"; do
    if brew list "$package" &> /dev/null; then
        echo "  ✅ $package already installed"
    else
        echo "  📦 Installing $package..."
        brew install "$package"
    fi
done

# Install Homebrew casks (GUI applications)
echo "📦 Installing Homebrew casks..."
CASKS=(
    ghostty
    raycast
    cursor
    docker
)

for cask in "${CASKS[@]}"; do
    if brew list --cask "$cask" &> /dev/null; then
        echo "  ✅ $cask already installed"
    else
        echo "  📦 Installing $cask..."
        brew install --cask "$cask"
    fi
done

# Install Rust via rustup
if command -v rustc &> /dev/null; then
    echo "✅ Rust already installed ($(rustc --version))"
else
    echo "🦀 Installing Rust via rustup..."
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
    source "$HOME/.cargo/env"
    echo "✅ Rust installed ($(rustc --version))"
fi

# Install oh-my-zsh
if [ -d "$HOME/.oh-my-zsh" ]; then
    echo "✅ oh-my-zsh already installed"
else
    echo "📦 Installing oh-my-zsh..."
    sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended
    echo "✅ oh-my-zsh installed"
fi

# Install zsh plugins
ZSH_CUSTOM="${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}"

# zsh-autosuggestions
if [ -d "$ZSH_CUSTOM/plugins/zsh-autosuggestions" ]; then
    echo "✅ zsh-autosuggestions already installed"
else
    echo "📦 Installing zsh-autosuggestions..."
    git clone https://github.com/zsh-users/zsh-autosuggestions "$ZSH_CUSTOM/plugins/zsh-autosuggestions"
    echo "✅ zsh-autosuggestions installed"
fi

# zsh-syntax-highlighting
if [ -d "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting" ]; then
    echo "✅ zsh-syntax-highlighting already installed"
else
    echo "📦 Installing zsh-syntax-highlighting..."
    git clone https://github.com/zsh-users/zsh-syntax-highlighting "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting"
    echo "✅ zsh-syntax-highlighting installed"
fi

# Install nvm
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

# Install Claude Code via npm
if command -v claude &> /dev/null; then
    echo "✅ Claude Code already installed"
else
    echo "🤖 Installing Claude Code..."
    npm install -g @anthropic-ai/claude-code
    echo "✅ Claude Code installed"
fi

echo ""
echo "✨ Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Run ./bootstrap.sh to deploy configurations"
echo "  2. Restart your shell or run: exec zsh"
