# dotfiles

A batteries-included dotfiles repository that turns a fresh macOS machine into a
fast, robust local development environment in a couple of commands. One run
installs the tools, the next deploys the configuration — no manual copy/paste,
no remembering which `brew install` flags you used last time.

## Why use this

- **Fast onboarding** — go from a blank Mac to a working terminal, shell,
  editor, Kubernetes tooling, and AI assistant in under 10 minutes.
- **Reproducible** — `Brewfile` is the single source of truth for tools, and
  GNU Stow handles configuration as transparent symlinks. No templating, no
  hidden state.
- **Idempotent and re-runnable** — every script is safe to run twice. Existing
  configs are backed up to a timestamped directory before anything is replaced.
- **Selective** — interactive menus let you pick which tools and configs to
  install. Use `--all` for unattended/CI installs.
- **Personalization without forking** — secrets like your git identity live in
  `~/.gitconfig.local` (untracked); shared preferences live in the repo. The
  same pattern applies to Claude Code settings via `.sample` templates that are
  merged into your local config.

## Overview

This repository contains personal dotfiles for macOS, managed with GNU Stow.
It includes configurations for:

- **Zsh** — shell with aliases, completions, and prompt customization
- **Git** — version control settings and global ignores
- **Vim** — minimal text editor configuration
- **Ghostty** — terminal emulator settings
- **k9s** — Kubernetes cluster management UI
- **Claude Code** — AI coding assistant: behavioral rules, permissions, hooks,
  statusline, sub-agents, and slash commands (see [Claude Code Configuration](#claude-code-configuration))
- **mise** — dev tools and environment manager

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
- gitleaks (Secret scanning for pre-commit + CI)
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

## Secret Scanning

This repository ships with [gitleaks](https://github.com/gitleaks/gitleaks) wired
into two layers to prevent accidental commits of tokens, API keys, or other
sensitive data:

- **Pre-commit hook** (`.githooks/pre-commit`) — scans staged changes with
  `gitleaks protect --staged --redact`. Enabled for this repo by
  `./install.sh` (item: `git-hooks`), which runs
  `git config core.hooksPath .githooks` locally — no global config is touched.
- **GitHub Actions** (`.github/workflows/gitleaks.yml`) — scans every push to
  `main` and every PR via the official `gitleaks/gitleaks-action`. Acts as a
  CI safety net for cases where the local hook is bypassed
  (e.g. `git commit --no-verify`).

If gitleaks reports a false positive, add an allowlist entry to a `.gitleaks.toml`
at the repo root. The default ruleset is used otherwise.

## Repository Structure

```
dotfiles/
├── .gitignore              # Excludes IDE and OS files
├── .stowrc                 # Stow configuration
├── .githooks/
│   └── pre-commit          # gitleaks scan on staged changes
├── .github/
│   └── workflows/
│       └── gitleaks.yml    # CI secret scan on push/PR
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
    │   └── .claude/
    │       ├── settings.sample.json    # Permissions, hooks, env, statusline
    │       ├── CLAUDE.sample.md        # Global behavioral rules
    │       ├── agents/                 # Specialized sub-agents
    │       │   ├── code-reviewer.md
    │       │   └── debugger.md
    │       ├── commands/               # Slash commands
    │       │   └── session-summary.md
    │       ├── hooks/                  # Lifecycle hooks
    │       │   ├── log-event.sh
    │       │   └── validate-bash.sh
    │       └── scripts/
    │           └── statusline.sh       # Custom statusline renderer
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

## Claude Code Configuration

[Claude Code](https://docs.claude.com/claude-code) is Anthropic's CLI for
AI-assisted development. This repo ships a complete, opinionated setup that
covers behavior, safety, observability, and workflow ergonomics.

### What gets installed

`./install.sh` installs the Claude Code CLI as a global npm package
(`@anthropic-ai/claude-code`). After install, configuration lives under
`~/.claude/`, which `./bootstrap.sh` populates from `configs/claude/.claude/`.

### How configuration is deployed

The Claude config uses a `.sample` template pattern so that the repository can
ship sensible defaults without overwriting your personal customizations:

| Repo file                          | Deployed as            | Behavior on re-run                                                |
| ---------------------------------- | ---------------------- | ----------------------------------------------------------------- |
| `.claude/settings.sample.json`     | `~/.claude/settings.json` | Deep-merged with your existing file — your settings win on conflict, arrays are deduplicated |
| `.claude/CLAUDE.sample.md`         | `~/.claude/CLAUDE.md`     | Created from template if missing; otherwise left alone           |
| `.claude/agents/*.md`              | `~/.claude/agents/*.md`   | Plain symlinks via Stow (live updates from the repo)             |
| `.claude/commands/*.md`            | `~/.claude/commands/*.md` | Plain symlinks via Stow                                          |
| `.claude/hooks/*.sh`               | `~/.claude/hooks/*.sh`    | Plain symlinks via Stow                                          |
| `.claude/scripts/*.sh`             | `~/.claude/scripts/*.sh`  | Plain symlinks via Stow                                          |

The merge logic for `settings.json` is implemented in `bootstrap.sh` using `jq`
— see the `*.sample.*` handling block. Your local edits to `settings.json`
survive future bootstrap runs.

### `settings.json` — what it configures

#### Environment variables

```json
"env": {
    "BASH_DEFAULT_TIMEOUT_MS": "120000",
    "CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY": "1",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
    "CLAUDE_CODE_FILE_READ_MAX_OUTPUT_TOKENS": "64000",
    "CLAUDE_CODE_MAX_OUTPUT_TOKENS": "64000",
    "DISABLE_TELEMETRY": "1"
}
```

- Generous output token budgets for long file reads.
- Telemetry and non-essential network traffic disabled.
- 2-minute default Bash timeout (overridable per call).

#### Top-level options

- `alwaysThinkingEnabled: true` — extended thinking is on for every turn.
- `includeCoAuthoredBy: false` — Claude does not add `Co-Authored-By` trailers
  to commits.
- `cleanupPeriodDays: 99999` — effectively disables transcript auto-cleanup so
  past sessions remain available locally.

#### Permissions — `allow` / `deny` / `ask`

Permissions follow the principle of least privilege:

- **`allow`** — fast, read-only/safe operations run without prompting: `git`,
  `gh <view|list>`, `kubectl get|describe|logs`, language toolchains (`npm`,
  `yarn`, `pnpm`, `cargo`, `go`), search/inspection tools (`rg`, `fd`, `jq`,
  `bat`), and `WebSearch`.
- **`deny`** — never allowed, even with confirmation. Covers secrets and shell
  rc files: `.env*`, `.envrc`, `.zshrc`, `.bashrc`, `.profile`, `.zsh_history`,
  `/etc/zprofile`, `/etc/zshrc`, `node_modules`, etc. Both Read and Write are
  blocked.
- **`ask`** — Claude must request confirmation before running. Covers
  destructive or remote-affecting commands: `rm`, `mv`, `cp`, `sudo`, `curl`,
  `wget`, `ssh`, `git push|rebase|reset|checkout|clean|branch -D`,
  `gh repo delete`, `gh secret set|delete`, `kubectl apply|delete`, and
  `docker`.

Edit `~/.claude/settings.json` directly to tighten or loosen these — your
changes are preserved on the next `./bootstrap.sh`.

#### Hooks

Hooks are shell commands the harness runs at specific lifecycle points. Two
hook scripts ship with this repo:

**`hooks/log-event.sh`** — universal event logger. Wired up to every supported
event:

| Event              | Purpose                                                      |
| ------------------ | ------------------------------------------------------------ |
| `SessionStart`     | Logged at the start of every session                         |
| `SessionEnd`       | Logged when the session ends                                 |
| `UserPromptSubmit` | Captures each user message                                   |
| `PreToolUse`       | Fires before any tool call (also: `Bash` validator below)    |
| `PostToolUse`      | Fires after any tool call                                    |
| `Stop`             | Fires when the assistant turn ends                           |
| `PreCompact`       | Before context auto-compaction                               |
| `PostCompact`      | After context auto-compaction                                |
| `Notification`     | When the harness needs your attention (e.g. permission ask)  |

Events are appended as JSONL to `$CLAUDE_PROJECT_DIR/.claude/hook-events.jsonl`
when invoked inside a project, otherwise `~/.claude/hook-events.jsonl`. Use
this for usage analytics, cost tracking, or post-hoc debugging:

```bash
jq -r 'select(.event == "PreToolUse") | .input.tool_name' \
    ~/.claude/hook-events.jsonl | sort | uniq -c | sort -rn
```

**`hooks/validate-bash.sh`** — a `PreToolUse` matcher for `Bash` calls. Blocks
commands that touch sensitive directories (`node_modules/`, `.env`, `.git/`,
`dist/`, `build/`, `.next/`, `.vscode/`, `.idea/`). Exits with status `2` to
prevent execution.

#### Statusline

```json
"statusLine": {
    "type": "command",
    "command": "~/.claude/scripts/statusline.sh"
}
```

`scripts/statusline.sh` renders a compact one-liner with:

- current directory and git branch
- active model
- session lines added/removed (colored)
- session duration (auto-scaled to ms/s/m/h)
- accumulated cost (only when > $0)
- a 15-cell context-window progress bar with percentage

Example:

```
/dotfiles (extend-claude-config) | claude-opus-4-7 +42 -7 in 3m for $0.18 | ████████░░░░░░░ 53%
```

### `CLAUDE.md` — global behavioral rules

`CLAUDE.md` is loaded into every Claude Code session as user-level system
instructions. The template (`CLAUDE.sample.md`) encodes preferences for:

- **Behavior** — direct, concise, no sycophancy, push back with technical
  reasons rather than agreeing reflexively.
- **Writing code** — smallest reasonable changes, match surrounding style, do
  not touch unrelated code or whitespace, never remove non-false comments.
- **Systematic debugging** — investigate root cause, never patch symptoms, one
  hypothesis at a time.
- **Version control** — concise imperative commit messages, frequent commits,
  WIP branches when ambiguous.
- **GitHub** — use `gh` high-level commands, never raw `gh api` calls.
- **Comments** — explain *why*, not *what*.

Edit `~/.claude/CLAUDE.md` to customize per-machine; project-specific
overrides go in `CLAUDE.md`/`AGENTS.md` at the project root.

### Sub-agents — `agents/`

Sub-agents are specialized personas with their own tool allowlist that Claude
can dispatch for focused tasks.

**`agents/code-reviewer.md`** — Senior code reviewer. Tools: `Read`, `Grep`,
`Glob`, `Bash`. Runs `git diff`, scans modified files, and returns feedback
organized by priority (critical / warning / suggestion) with concrete fixes.

**`agents/debugger.md`** — Root-cause debugger. Tools: `Read`, `Edit`, `Bash`,
`Grep`, `Glob`. Captures the failure, isolates the location, forms hypotheses,
and applies a minimal fix with a verification step.

Add new agents by dropping a Markdown file into `configs/claude/.claude/agents/`
with frontmatter (`name`, `description`, `tools`) followed by the system
prompt body.

### Slash commands — `commands/`

**`/session-summary`** — generates `session_{slug}_{timestamp}.md` containing
key actions, total cost, efficiency insights, possible process improvements,
turn count, and observations. Useful for keeping a personal log of long
sessions.

Add new commands by dropping a Markdown file into
`configs/claude/.claude/commands/`. The filename (minus `.md`) becomes the
slash command name.

### Customizing your setup

```bash
# Edit settings (merged on next bootstrap; your edits win)
vim ~/.claude/settings.json

# Add a project-specific override (loaded for that project only)
echo "Use Python 3.12 conventions." > /path/to/project/CLAUDE.md

# Add a new sub-agent (tracked in this repo)
vim configs/claude/.claude/agents/security-auditor.md

# Add a new slash command (tracked in this repo)
vim configs/claude/.claude/commands/release-notes.md

# Re-deploy
./bootstrap.sh
```

### Inspecting hook activity

```bash
# Tail events live
tail -f ~/.claude/hook-events.jsonl | jq .

# Count tool calls in the current project
jq -r 'select(.event == "PostToolUse") | .input.tool_name' \
    .claude/hook-events.jsonl | sort | uniq -c
```

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
