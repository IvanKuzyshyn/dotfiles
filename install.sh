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

# Install nvm
if [ -d "$HOME/.nvm" ]; then
    echo "✅ nvm already installed"
else
    echo "📦 Installing nvm..."
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
    export NVM_DIR="$HOME/.nvm"
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
    echo "✅ nvm installed"
fi

echo ""
echo "✨ Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Run ./bootstrap.sh to deploy configurations"
echo "  2. Restart your shell or run: exec zsh"
