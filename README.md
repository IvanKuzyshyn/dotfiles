# dotfiles

Personal configuration files for quick terminal and tool setup. Simple, maintainable, and ready to deploy.

## Overview

This repository contains my personal dotfiles for macOS, managed with GNU Stow. It includes configurations for:

- **Zsh** - Shell with aliases, completions, and prompt customization
- **Git** - Version control settings and global ignores
- **Vim** - Minimal text editor configuration
- **Ghostty** - Terminal emulator settings
- **k9s** - Kubernetes cluster management UI
- **Claude** - AI assistant settings
- **mise** - Dev tools and environment manager

## Quick Start

```bash
# Clone the repository
git clone git@github.com:IvanKuzyshyn/dotfiles.git ~/dotfiles
cd ~/dotfiles

# Install tools
./install.sh

# Deploy configurations
./bootstrap.sh

# Restart your shell
exec zsh
```

Both scripts present an interactive selection menu — toggle items on/off before proceeding. Pass `--all` to skip the menu and select everything (useful for CI or scripted setups).

## Installation

### 1. Install Tools

The `install.sh` script installs all required tools:

```bash
./install.sh          # interactive menu
./install.sh --all    # non-interactive, install everything
```

**Homebrew packages and casks** (managed via `Brewfile`):
- GNU Stow (config management)
- k9s, kubectl (Kubernetes)
- awscli, gh (Cloud & GitHub)
- jq, yq (JSON/YAML processors)
- ncdu (Disk usage analyzer)
- git, go (Development)
- mise (Dev tools & env manager)
- Ghostty (Terminal emulator)
- Raycast (Productivity launcher)
- Cursor (AI code editor)
- Docker (Docker Desktop)

**Other tools** (dedicated installers):
- Rust (via rustup)
- oh-my-zsh with plugins (zsh-autosuggestions, zsh-syntax-highlighting)
- nvm + Node.js LTS
- Claude Code (via npm)

### 2. Deploy Configurations

The `bootstrap.sh` script creates symlinks using GNU Stow:

```bash
./bootstrap.sh          # interactive menu
./bootstrap.sh --all    # non-interactive, deploy everything
```

**What happens:**
- Auto-detects configs from `configs/*/` directories
- Prompts for your git name and email (creates `~/.gitconfig.local`)
- Backs up existing configs to `~/.dotfiles_backup_<timestamp>`
- Creates symlinks from this repo to your home directory
- Shows verbose output of what's being linked

### 3. Verify

Check that symlinks are created:

```bash
ls -la ~ | grep "^l"
```

You should see symlinks for `.zshrc`, `.gitconfig`, `.vimrc`, etc.

## Repository Structure

```
dotfiles/
├── .gitignore              # Excludes IDE and OS files
├── .stowrc                 # Stow configuration
├── Brewfile                # Homebrew packages and casks (single source of truth)
├── install.sh              # Tool installation script
├── bootstrap.sh            # Config deployment script
├── README.md               # This file
├── AGENTS.md               # AI agent guidance
├── lib/
│   └── menu.sh             # Shared interactive selection menu library
└── configs/                # Tool configurations (stow packages)
    ├── zsh/
    │   └── .zshrc             # Shell configuration
    ├── git/
    │   ├── .gitconfig         # Git settings
    │   └── .gitignore_global  # Global Git ignores
    ├── vim/
    │   └── .vimrc             # Vim configuration
    ├── ghostty/
    │   └── .config/ghostty/config
    ├── k9s/
    │   └── .config/k9s/config.yaml
    ├── claude/
    │   └── .claude/settings.json
    └── mise/
        └── .config/mise/config.toml
```

## Features

### Zsh Configuration

- **Aliases**: Kubernetes (`k`, `kgp`), Git (`gs`, `gco`), and directory shortcuts
- **Completions**: kubectl, AWS CLI, GitHub CLI
- **Prompt**: Git branch integration with color coding
- **Tool Integration**: nvm, Rust (cargo)

### Git Configuration

- **User**: Personalized during bootstrap (stored in `~/.gitconfig.local`)
- **Aliases**: Common shortcuts (`st`, `co`, `lg`)
- **Global Ignores**: OS files, IDE configs, language artifacts

The git configuration uses an include pattern to separate personal identity
from shared preferences. Your name and email are stored in `~/.gitconfig.local`,
which is not tracked in the repository.

### Kubernetes Tools

- **kubectl aliases**: Speed up cluster management
- **k9s config**: UI preferences and performance settings

### mise

- **Global config**: Auto-installs missing tools, uses precompiled binaries
- **Per-project**: Drop a `mise.toml` in any project for tool version management

## Adding New Tools

### Homebrew package or cask

Add one line to `Brewfile`:

```ruby
brew "newtool"       # CLI package
cask "newapp"        # GUI application
```

Then run `./install.sh` and select "homebrew-tools".

### Configuration

1. Create a new directory in `configs/`:
   ```bash
   mkdir -p configs/newtool
   ```

2. Add config files (will be symlinked to `$HOME`):
   ```bash
   configs/newtool/.newtoolrc
   ```

3. Deploy (the config is auto-detected — no script changes required):
   ```bash
   ./bootstrap.sh
   ```

## Troubleshooting

### Stow conflicts

The bootstrap script detects conflicts automatically and offers to back up or remove existing files. If you need to resolve manually:

```bash
# Remove existing configs (they're backed up)
rm ~/.zshrc ~/.gitconfig ~/.vimrc

# Try again
./bootstrap.sh
```

### Missing tools

```bash
# Install individually
brew install <tool-name>

# Or re-run the full install
./install.sh
```

### Broken symlinks

Check for broken links:

```bash
find ~ -maxdepth 1 -type l ! -exec test -e {} \; -print
```

Remove and restow if needed:

```bash
cd ~/dotfiles
stow -D zsh  # Unstow
stow zsh     # Restow
```

## Contributing

This is a personal dotfiles repository, but you're welcome to:

- **Fork and customize** - Use this as a starting point for your own setup
- **Share ideas** - Open an issue to suggest improvements or share configurations
- **Report issues** - Let me know if something doesn't work as expected

### Using as a Template

```bash
# Fork the repository
# Clone your fork
git clone git@github.com:yourusername/dotfiles.git ~/dotfiles
cd ~/dotfiles

# Customize configurations to your preferences
vim configs/git/.gitconfig
vim configs/zsh/.zshrc

# Deploy your customized configs
./bootstrap.sh
```

## License

Public domain via [Unlicense](LICENSE). Use freely.
