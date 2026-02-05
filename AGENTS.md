# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## Repository Overview

This is a personal dotfiles repository for managing user configuration files across macOS systems. It uses GNU Stow for symlink management and Homebrew for tool installation.

**Architecture Philosophy:**
- Simple and maintainable over complex abstractions
- Transparent symlink management (no templates)
- Monolithic configuration files (single .zshrc)
- Tool-centric directory structure

## Repository Structure

```
dotfiles/
├── .gitignore              # Excludes IDE/OS files
├── .stowrc                 # Stow configuration (target=$HOME)
├── install.sh              # Tool installation via Homebrew
├── bootstrap.sh            # Config deployment via Stow
├── README.md               # User-facing documentation
├── AGENTS.md               # This file
├── LICENSE                 # Unlicense (public domain)
├── zsh/
│   └── .zshrc             # Shell config (~150 lines)
├── git/
│   ├── .gitconfig         # Git settings
│   └── .gitignore_global  # Global ignores
├── vim/
│   └── .vimrc             # Minimal Vim config
├── ghostty/
│   └── .config/ghostty/config
└── k9s/
    └── .config/k9s/config.yaml
```

## Git Information

- **Remote**: git@github.com:IvanKuzyshyn/dotfiles.git
- **Main branch**: main
- **Author**: Ivan Kuzyshyn <kuzyshyn.ivan@gmail.com>
- **License**: Public domain (Unlicense)

## Setup Workflow

The repository provides a two-step setup process:

1. **`./install.sh`** - Installs tools via Homebrew
   - Detects platform (macOS only currently)
   - Installs Homebrew if missing
   - Installs packages: stow, k9s, kubectl, awscli, gh, jq, yq, ncdu, git, go
   - Installs casks: ghostty (terminal emulator), raycast (productivity launcher)
   - Installs Rust via rustup
   - Installs nvm
   - Idempotent (safe to re-run)

2. **`./bootstrap.sh`** - Deploys configurations via Stow
   - Prompts for git user name and email
   - Creates `~/.gitconfig.local` with personal git identity
   - Backs up existing configs to timestamped directory
   - Creates symlinks for each tool directory
   - Uses verbose mode for transparency
   - Lists backup location on completion

## Tool Installation Details

| Tool | Method | Location |
|------|--------|----------|
| Homebrew packages | `brew install` | Managed by Homebrew |
| Homebrew casks (GUI apps) | `brew install --cask` | Managed by Homebrew |
| Ghostty, Raycast | `brew install --cask` | `/Applications/` |
| Rust | rustup installer | `~/.cargo/` |
| nvm | Official installer | `~/.nvm/` |
| GNU Stow | `brew install stow` | Required for deployment |

## Configuration Files

### Zsh (`zsh/.zshrc`)

**Structure (in order):**
1. Environment variables (EDITOR, LANG, etc.)
2. Path configuration (Go, Homebrew, local bin)
3. nvm setup
4. Rust setup
5. Aliases (Kubernetes, Git, directory)
6. Tool completions (AWS CLI, gh, kubectl)
7. Prompt (git branch via vcs_info)
8. History settings
9. Completion system
10. Key bindings

**Key features:**
- Kubernetes aliases (`k=kubectl`, `kgp`, `kgs`, etc.)
- Git aliases (`gs`, `gco`, `gb`, etc.)
- Directory shortcuts (`ll`, `la`, `..`)
- Shell completions for tools
- Git branch in prompt

### Git (`git/`)

**`.gitconfig`:**
- User: Configured via `~/.gitconfig.local` (not in repo)
- Include directive: References `~/.gitconfig.local` for personal settings
- Editor: vim
- Default branch: main
- Aliases: st, co, br, ci, lg (graph log)
- Color settings enabled

**`.gitconfig.local`** (not in repository):
- Created during bootstrap with user's name and email
- Location: `~/.gitconfig.local`
- Included by main .gitconfig via `[include]` directive
- Contains only `[user]` section with personal identity

**`.gitignore_global`:**
- OS files (.DS_Store, Thumbs.db)
- IDE configs (.idea/, .vscode/, *.swp)
- Language artifacts (node_modules/, __pycache__/, target/)
- Tools (.terraform/, .claude/)

### Vim (`vim/.vimrc`)

**Minimal configuration:**
- Line numbers (number, relativenumber)
- Search (incsearch, hlsearch, ignorecase, smartcase)
- Indentation (autoindent, smartindent, tabstop=4, expandtab)
- Basic UI (ruler, showcmd, cursorline)
- No plugins (user doesn't use Vim heavily)

### Ghostty (`ghostty/.config/ghostty/config`)

**Basic terminal settings:**
- Font: JetBrains Mono, size 14
- Theme: dark:Dracula
- Window padding: 10px
- Cursor: block, no blink
- Shell integration: zsh
- macOS-specific options

### k9s (`k9s/.config/k9s/config.yaml`)

**Kubernetes UI configuration:**
- Refresh rate: 2 seconds
- Read-only: false
- UI: mouse disabled, dark skin
- Shell pod: busybox:1.35.0

## Personalization

### Git Identity

The repository uses git's `[include]` directive to separate personal identity
from shared configuration:

- **Shared config**: `git/.gitconfig` (tracked in repo)
  - Contains all git preferences, aliases, and settings
  - Does NOT contain user name/email

- **Personal config**: `~/.gitconfig.local` (not tracked)
  - Created during `./bootstrap.sh`
  - Contains only `[user]` section with name and email
  - Never committed to the repository

**To update your git identity:**
```bash
# Option 1: Re-run bootstrap
./bootstrap.sh

# Option 2: Edit directly
vim ~/.gitconfig.local

# Option 3: Use git commands
git config --file ~/.gitconfig.local user.name "Your Name"
git config --file ~/.gitconfig.local user.email "your.email@example.com"
```

**To add more personal settings:**
You can add any git configuration to `~/.gitconfig.local`. For example:
```
[user]
    name = Your Name
    email = your.email@example.com
    signingkey = YOUR_GPG_KEY

[commit]
    gpgsign = true
```

## Adding New Tools

To add a new tool configuration:

1. Create directory: `mkdir -p newtool/`
2. Add config files (will be symlinked to `$HOME`)
3. Update `bootstrap.sh` TOOLS array
4. Run `./bootstrap.sh` to deploy

**Example:**
```bash
# Add tmux
mkdir -p tmux/
echo "set -g mouse on" > tmux/.tmux.conf

# Edit bootstrap.sh, add "tmux" to TOOLS array

# Deploy
./bootstrap.sh
```

## Maintenance Guidelines

### Modifying Configurations

1. Edit files in the dotfiles repo
2. Changes are immediately reflected (symlinked)
3. Commit and push changes

### Script Modifications

**`install.sh`:**
- Add new tools to PACKAGES array
- Maintain idempotency (check if already installed)
- Keep platform detection logic

**`bootstrap.sh`:**
- Add new tool directories to TOOLS array
- Maintain backup logic
- Keep verbose output

### Testing Changes

Before committing:
1. Test syntax: `bash -n script.sh`
2. Test stowing: `stow -n -v newtool` (dry run)
3. Verify symlinks: `ls -la ~/ | grep newtool`

## Platform Support

**Currently supported:**
- macOS (Darwin)

**Future considerations:**
- Linux (apt/dnf detection in install.sh)
- Platform-specific configurations

## Common Issues

### Stow Conflicts

**Cause:** Existing files at target locations
**Solution:** Backup and remove, or run bootstrap.sh (automatic backup)

### Missing Homebrew

**Cause:** First time on new system
**Solution:** install.sh installs Homebrew automatically

### Broken Symlinks

**Cause:** Dotfiles directory moved
**Solution:** Unstow and restow from new location

```bash
cd ~/dotfiles
stow -D zsh  # Unstow
stow zsh     # Restow
```

## Development Notes

- `.gitignore` excludes `.idea/`, `.DS_Store`, `.vscode/`, `*.swp`, `*.log`
- Scripts are executable (`chmod +x install.sh bootstrap.sh`)
- `.stowrc` configures target directory and ignore patterns
- Backups are timestamped: `~/.dotfiles_backup_YYYYMMDD_HHMMSS`

## Future Enhancements (Out of Scope)

- Linux platform support
- tmux configuration
- SSH config management
- macOS defaults automation
- Update script (git pull + restow)
- Secret management integration
