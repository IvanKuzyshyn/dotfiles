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
├── .stowrc                 # Stow configuration (dir=configs, target=$HOME)
├── .githooks/
│   └── pre-commit          # gitleaks secret scan on staged changes
├── .github/
│   └── workflows/
│       └── gitleaks.yml    # CI secret scan on push/PR
├── Brewfile                # Homebrew packages and casks (single source of truth)
├── install.sh              # Tool installation via Homebrew
├── bootstrap.sh            # Config deployment via Stow
├── README.md               # User-facing documentation
├── AGENTS.md               # This file
├── LICENSE                 # Unlicense (public domain)
├── lib/
│   └── menu.sh             # Shared confirmation prompt library
└── configs/                # Tool configurations (stow packages)
    ├── zsh/
    │   └── .zshrc             # Shell config (~150 lines)
    ├── git/
    │   ├── .gitconfig         # Git settings
    │   └── .gitignore_global  # Global ignores
    ├── vim/
    │   └── .vimrc             # Minimal Vim config
    ├── ghostty/
    │   └── .config/ghostty/config
    ├── k9s/
    │   └── .config/k9s/config.yaml
    └── mise/
        └── .config/mise/config.toml
```

## Setup Workflow

The repository provides a two-step setup process:

1. **`./install.sh`** - Installs tools via Homebrew
   - Detects platform (macOS only currently)
   - Prompts `[Y/n]` for each tool (Enter = yes, `n` = skip)
   - Installs Homebrew if missing
   - Installs packages: stow, k9s, kubectl, kubectx, helm, kind, ko, oras, kluctl, awscli, gh, jq, yq, ncdu, git, go, mise, gitleaks
   - Installs casks: ghostty (terminal emulator), raycast (productivity launcher), cursor (AI code editor), docker (Docker Desktop)
   - Installs Rust via rustup
   - Installs oh-my-zsh with plugins (zsh-autosuggestions, zsh-syntax-highlighting)
   - Installs nvm and Node.js LTS
   - Installs Claude Code via npm
   - Configures this repo's pre-commit hook (`core.hooksPath=.githooks`) for gitleaks scanning
   - Idempotent (safe to re-run)
   - `--all` flag skips prompts (installs everything)

2. **`./bootstrap.sh`** - Deploys configurations via Stow
   - Prompts `[Y/n]` for each config (Enter = yes, `n` = skip)
   - For each confirmed tool: checks conflicts → git config (if git) → stow
   - Prompts for git user name and email when deploying git config
   - Backs up existing configs to timestamped directory
   - Creates symlinks for each tool directory
   - Uses verbose mode for transparency
   - Lists backup location on completion
   - `--all` flag skips tool prompts
   - `--backup` flag auto-backs-up conflicts without prompting

## Tool Installation Details

| Tool | Method | Location |
|------|--------|----------|
| Homebrew packages | `brew install` | Managed by Homebrew |
| Homebrew casks (GUI apps) | `brew install --cask` | Managed by Homebrew |
| Ghostty, Raycast, Cursor, Docker | `brew install --cask` | `/Applications/` |
| Rust | rustup installer | `~/.cargo/` |
| oh-my-zsh | Official installer | `~/.oh-my-zsh/` |
| zsh plugins | git clone | `~/.oh-my-zsh/custom/plugins/` |
| nvm | Official installer | `~/.nvm/` |
| Node.js | nvm | `~/.nvm/versions/node/` |
| Claude Code | npm global | `~/.npm/` or nvm node path |
| mise | `brew install mise` | Dev tools & env manager |
| GNU Stow | `brew install stow` | Required for deployment |
| gitleaks | `brew install gitleaks` | Secret scanning (pre-commit + CI) |

## Configuration Files

### Zsh (`configs/zsh/.zshrc`)

**Structure (in order):**
1. Oh My Zsh configuration and plugins
2. Environment variables (EDITOR, LANG, etc.)
3. Path configuration (Go, Homebrew, local bin)
4. nvm setup
5. Rust setup
6. Aliases (Kubernetes, Git, directory)
7. Tool completions (AWS CLI, gh, kubectl)
8. Prompt (git branch via vcs_info)
9. History settings
10. Completion system
11. Key bindings

**Oh My Zsh plugins:**
- zsh-autosuggestions (fish-like autosuggestions)
- zsh-syntax-highlighting (syntax highlighting for commands)

**Key features:**
- Kubernetes aliases (`k=kubectl`, `kgp`, `kgs`, etc.)
- Git aliases (`gs`, `gco`, `gb`, etc.)
- Directory shortcuts (`ll`, `la`, `..`)
- Shell completions for tools
- Git branch in prompt

### Git (`configs/git/`)

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

### Vim (`configs/vim/.vimrc`)

**Minimal configuration:**
- Line numbers (number, relativenumber)
- Search (incsearch, hlsearch, ignorecase, smartcase)
- Indentation (autoindent, smartindent, tabstop=4, expandtab)
- Basic UI (ruler, showcmd, cursorline)
- No plugins (user doesn't use Vim heavily)

### Ghostty (`configs/ghostty/.config/ghostty/config`)

**Basic terminal settings:**
- Font: JetBrains Mono, size 14
- Theme: Dracula+
- Window padding: 10px
- Cursor: block, no blink
- Shell integration: zsh
- macOS-specific options

### k9s (`configs/k9s/.config/k9s/config.yaml`)

**Kubernetes UI configuration:**
- Refresh rate: 2 seconds
- Read-only: false
- UI: mouse disabled, dark skin
- Shell pod: busybox:1.35.0

### mise (`configs/mise/.config/mise/config.toml`)

**Global dev tools and environment manager:**
- auto_install: automatically installs missing tools when entering a directory
- Uses precompiled binaries when available
- 8 parallel jobs for tool installation
- Per-project tool versions via `mise.toml` in project roots

## Personalization

### Git Identity

The repository uses git's `[include]` directive to separate personal identity
from shared configuration:

- **Shared config**: `configs/git/.gitconfig` (tracked in repo)
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

### Add a Homebrew package or cask

Add one line to `Brewfile`:

```ruby
brew "newtool"       # CLI package
cask "newapp"        # GUI application
```

Then run `./install.sh` and confirm when prompted for "homebrew-tools".

### Add a non-brew tool

Add an `add_item` call and install block in `install.sh`.

### Add a new configuration

1. Create directory: `mkdir -p configs/newtool/`
2. Add config files (will be symlinked to `$HOME`)
3. (Optional) Add a description to `get_tool_desc()` in `bootstrap.sh`
4. Run `./bootstrap.sh` to deploy (conflicts are auto-detected via stow dry run)

The config is auto-detected from `configs/*/` — no array to update.

**Example:**
```bash
# Add tmux config
mkdir -p configs/tmux/
echo "set -g mouse on" > configs/tmux/.tmux.conf

# Deploy (tmux will appear in the prompt list automatically)
./bootstrap.sh
```

## Secret Scanning

This repo runs `gitleaks` in two places to prevent accidental commits of tokens, API keys, or other sensitive data:

- **Pre-commit hook** — `.githooks/pre-commit` runs `gitleaks protect --staged --redact --verbose` and blocks the commit on any finding. Enabled per-repo via `git config core.hooksPath .githooks`, set by `./install.sh` (item: `git-hooks`).
- **GitHub Actions** — `.github/workflows/gitleaks.yml` runs on every push to `main` and every PR. CI safety net in case the local hook is bypassed (e.g. `git commit --no-verify`).

The default gitleaks ruleset is used (no `.gitleaks.toml`). Add allowlists or custom rules to a `.gitleaks.toml` at the repo root if false positives appear.

**Important:** scope is repo-local only. The `core.hooksPath` setting is written to this repo's `.git/config`, never to the global gitconfig.

## Maintenance Guidelines

### Modifying Configurations

1. Edit files in the dotfiles repo
2. Changes are immediately reflected (symlinked)
3. Commit and push changes

### Script Modifications

**`Brewfile`:**
- Add/remove brew packages and casks here (single source of truth)
- `brew bundle` handles idempotency natively

**`install.sh`:**
- Sources `lib/menu.sh` for confirmation prompts
- Brew tools managed via `Brewfile` (not arrays in install.sh)
- Non-brew tools (rust, oh-my-zsh, nvm, claude-code) have dedicated install blocks
- `git-hooks` item runs `git config core.hooksPath .githooks` (scoped to this repo only)
- Each tool confirmed inline via `confirm_item` before its install block
- Supports `--all` flag to skip prompts (install everything)

**`bootstrap.sh`:**
- Sources `lib/menu.sh` for confirmation prompts
- Auto-detects configs from `configs/*/` directories
- Single per-tool loop: confirm → check conflicts → git config → stow
- Supports `--all` flag to skip tool prompts
- Supports `--backup` flag to auto-backup conflicts without prompting
- `--all --backup` together = fully non-interactive (CI mode)

**`lib/menu.sh`:**
- Shared confirmation prompt library (do not duplicate in scripts)
- Bash 3.2 compatible (no associative arrays, no namerefs)
- Key functions: `parse_flags`, `add_item`, `confirm_item`
- Flags: `--all` (sets `NON_INTERACTIVE`), `--backup` (sets `AUTO_BACKUP`)

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
- `.stowrc` configures stow directory (`configs/`), target directory, and ignore patterns
- Backups are timestamped: `~/.dotfiles_backup_YYYYMMDD_HHMMSS`

## Future Enhancements (Out of Scope)

- Linux platform support
- tmux configuration
- SSH config management
- macOS defaults automation
- Update script (git pull + restow)
- Secret management integration
