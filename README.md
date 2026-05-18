# dotfiles

A personal dotfiles repository for macOS (and, where possible, Linux) managed
by a Go binary, `dot`. The binary installs tools and deploys configuration
files using embedded YAML manifests — one manifest per tool — so the same
machine setup is reproducible from a fresh shell.

The `configs/` directory holds the actual user-facing config files (`.zshrc`,
`.gitconfig`, etc.). The `tools/` directory holds manifests describing how to
install each tool and where its configs go. `dot` reads both, validates them,
resolves dependencies, and runs the work in a stable order with per-tool log
streams.

## Install

One-line curl bootstrap (downloads the latest release tarball, verifies its
SHA-256 checksum, and installs `dot` to `$HOME/.local/bin`):

```bash
curl -fsSL https://github.com/ivankuzyshyn/dotfiles/releases/latest/download/install.sh | sh
```

Set `PREFIX` to install elsewhere (e.g. `PREFIX=/usr/local/bin`).

Alternatively, build from source:

```bash
git clone git@github.com:ivankuzyshyn/dotfiles.git ~/dotfiles
cd ~/dotfiles
make build
./bin/dot
```

## Usage

`dot` is a single binary with a small set of subcommands.

| Command | Purpose |
| --- | --- |
| `dot` (no args, in a TTY) | Launch the Bubble Tea picker |
| `dot list` | Enumerate known tools |
| `dot install [tool...]` | Install tools and their dependencies (alias: `dot update`) |
| `dot deploy [tool...]` | Link config files into `$HOME` (no install steps) |
| `dot status [tool...]` | Report which tools are installed and deployed |
| `dot version` | Print the embedded version |

`install` and `deploy` accept tool names as positional arguments, plus
`--all` to select every known tool and `--tag <tag>` to select by manifest
tag. `--no-deps` skips dependency expansion.

### Global flags

- `--non-interactive` — never open the TUI, never prompt; required for CI.
- `--config-dir <dir>` — overlay manifests directory (default
  `~/.config/dot`). Files here are loaded on top of the embedded set.
- `--dotfiles-dir <dir>` — root of this repo (default: auto-detected from
  cwd, `DOTFILES_DIR`, or `~/dotfiles`).
- `-v`, `--verbose` — stream log lines to stderr as they happen.
- `--on-conflict=backup|overwrite|skip|abort` — config-deploy conflict
  policy (default `abort`). Applies to `install` and `deploy`.

### Per-run logs

Every run writes a structured event log to
`~/.local/state/dot/runs/<UTC-timestamp>.log`. The end-of-run summary
prints the path. The directory is capped at 10 recent runs.

## Adding a tool

Drop a YAML manifest in `tools/` (embedded into the binary at build time) or
in `~/.config/dot/tools/` (loaded as a user overlay; useful for per-machine
extras you do not want to commit).

```yaml
name: mytool
description: Example
platforms: [darwin, linux]
depends_on: [homebrew]
tags: [cli]
steps:
  - type: shell
    name: install
    check: 'command -v mytool'
    install: |
      brew install mytool
configs:
  - source: mytool
    target: ~
```

Available step `type` values:

- `shell` — arbitrary shell with optional `check` (skip-if-true) and
  `install` (the work).
- `homebrew_bootstrap` — install Homebrew itself if missing.
- `brew_package` — `brew install <pkg>`.
- `brew_cask` — `brew install --cask <pkg>`.
- `brewfile` — `brew bundle --file=<path>` for batch installs.
- `git_clone` — clone or update a repo into a target directory.
- `npm_global` — `npm install -g <pkg>`.
- `git_config` — set `git config` keys (scoped to this repo or globally).

`configs` is a list of `{ source, target }` pairs. `source` is a subdirectory
under `configs/`; `target` is where in `$HOME` it lives. The linker walks
every file under `source` and recreates the relative directory structure
under `target`, creating intermediate directories as needed.

## Configuration files

The `configs/` directory holds the files that get symlinked into `$HOME`:

- `configs/git/` — `.gitconfig`, `.gitignore_global` → `$HOME/`
- `configs/zsh/` — `.zshrc` → `$HOME/.zshrc`
- `configs/vim/` — `.vimrc` → `$HOME/.vimrc`
- `configs/ghostty/.config/ghostty/config` → `~/.config/ghostty/config`
- `configs/k9s/.config/k9s/config.yaml` → `~/.config/k9s/config.yaml`
- `configs/mise/.config/mise/config.toml` → `~/.config/mise/config.toml`
- `configs/claude/.claude/...` → `~/.claude/...`

Nested paths under `source` are preserved verbatim under `target`. For
example, `configs/ghostty/.config/ghostty/config` with `target: ~` produces
`~/.config/ghostty/config`.

The shipped `configs/git/.gitconfig` uses `[include] path = ~/.gitconfig.local`
so personal identity (`user.name`, `user.email`, GPG keys) stays in
`~/.gitconfig.local` and out of the repository.

## Tooling

### Makefile

| Target | Effect |
| --- | --- |
| `make build` | Builds `bin/dot` with the version stamp baked in via `-ldflags`. |
| `make test` | Runs unit tests (`go test ./...`). |
| `make test-int` | Runs integration tests (build tag `integration`). |
| `make test-snapshot` | Runs TUI snapshot tests against committed goldens (build tag `teatest`). |
| `make snapshot` | Regenerates the TUI snapshot goldens. Use sparingly; review the diff before committing. |
| `make lint` | Runs `golangci-lint`. |
| `make fmt` | Formats with `gofumpt` and `goimports`. |
| `make fmt-check` | Exits non-zero if formatting drift exists. |
| `make tidy` | `go mod tidy`. |
| `make clean` | Removes `bin/` and `dist/`. |

### CI

- `.github/workflows/ci.yml` — runs build, test, lint, fmt-check, and the
  TUI snapshot suite on every push and pull request.
- `.github/workflows/gitleaks.yml` — scans every push to `main` and every
  PR for committed secrets via the official `gitleaks/gitleaks-action`.

### Pre-commit hook

`.githooks/pre-commit` runs `gitleaks protect --staged --redact --verbose`
on staged changes. Enable it per-clone with:

```bash
git config core.hooksPath .githooks
```

This setting is scoped to this repository's `.git/config`; nothing global is
touched.

## Project structure

Go source lives in `cmd/dot/` and `internal/...`. The design spec and the
phased implementation plan live in `docs/superpowers/`.

## License

Public domain via [Unlicense](LICENSE). Use freely.
