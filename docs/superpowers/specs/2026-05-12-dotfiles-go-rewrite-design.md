# Dotfiles Go Rewrite Design

**Date:** 2026-05-12
**Status:** Approved
**Author:** Ivan Kuzyshyn (with Claude)

## Goal

Replace the current Bash-based dotfiles installer (`install.sh`, `bootstrap.sh`, GNU Stow) with a single Go binary named `dot` that provides a Bubble Tea TUI for installing tools, deploying configs, and resolving conflicts. Same dotfiles repo, new runner.

## Objectives

Functional:
- Self-contained binaries for macOS and Linux (no packages required to run `dot` itself).
- Easy to add a new tool via a YAML manifest, with no rebuild required for personal additions.
- Install everything from scratch, install/update only specific tools.
- Fault tolerance: a failed tool does not abort the run; independent tools continue.
- Supports tools that bring config files (`zsh`, `git`, `vim`, `ghostty`, `k9s`, `mise`, future `claude`, etc.), including conflict resolution when target files exist.
- Good UX with a strong feedback loop during installs and deploys.

Non-functional:
- Easy to write unit and integration tests.
- Good local + CI setup for tests, lint, and formatting.

## Decisions (with rationale)

| # | Decision | Choice | Rationale |
|---|----------|--------|-----------|
| 1 | Installation backend | Orchestrator on top of existing installers (`brew`, `rustup`, `npm`, `git`, `curl \| sh`) | The new tool's value is UX, extensibility, fault tolerance, testability — not reimplementing package installation. The binary itself is self-contained; runtime installers are bootstrapped on first run. |
| 2 | Invocation model | TUI-first with CLI escape hatches | Default `dot` opens an interactive picker with live progress and inline conflict resolution. Subcommands cover scripting/CI. |
| 3 | Extensibility model | Declarative YAML manifest with `shell` escape hatch | Most tools are clean step compositions; oddballs (`oh-my-zsh`, `rustup`, `nvm`) drop into shell without forcing everything into a perfect step taxonomy. |
| 4 | Linux scope | macOS-first; Linux binary cross-compiles but tool manifests can declare `platforms:`. Linux backends added incrementally when needed. | No current Linux usage. Building cross-platform now is speculative. Data model is forward-compatible. |
| 5 | Symlink management | Native Go; keep existing `configs/<tool>/` layout unchanged | Stow's conflict UX is awkward via stderr parsing. Native code gives TUI-integrated conflict prompts and is easy to unit-test. |
| 6 | Manifest location | Embedded defaults + external overlay (`$DOTFILES_DIR/tools/*.yaml`, `~/.config/dot/tools/*.yaml`) | Useful out of the box; trivially extensible per-machine; same-name overlay overrides embedded. |
| 7 | Update semantics | Reconcile model. `install` and `update` are aliases. Separate `dot status` calls `Check()` only. | Composes naturally with the `Step` interface (`Check()` + `Run()`). Avoids state-tracking complexity. |
| 8 | Repo strategy | Replace in-place. Build alongside Bash in `cmd/dot/`, delete Bash scripts and `.stowrc` at cutover. | One repo, one identity. Avoids parallel-implementation rot. |
| 9 | Distribution | GitHub Releases via Goreleaser + curl bootstrap script. Homebrew tap deferred. | Lowest-friction publishing path. Brew tap can be added later from the same Goreleaser pipeline. |
| 10 | TUI framework / config format | Bubble Tea + Lip Gloss + Bubbles, YAML manifests | De-facto Go TUI stack; YAML is more common for declarative tool configs and supports comments. |

## Architecture

### Repo layout

```
dotfiles/
├── cmd/dot/main.go              # Cobra root command
├── internal/
│   ├── cli/                     # Subcommands (install, update, status, deploy, list, version)
│   ├── manifest/                # YAML loading, schema, validation
│   ├── tool/                    # Tool model, registry (embedded + overlay merge), topological sort
│   ├── step/                    # Step interface + builtin implementations
│   ├── runner/                  # Orchestrator, event emission, fault tolerance
│   ├── linker/                  # Native symlink management
│   ├── platform/                # OS detection, platform-specific helpers
│   └── tui/                     # Bubble Tea models, views, components
├── tools/                       # Embedded default manifests (*.yaml)
├── configs/                     # UNCHANGED — existing config files for each tool
├── .goreleaser.yaml             # Release pipeline
├── .golangci.yaml               # Linter config
├── Makefile                     # build/test/lint/fmt/release targets
├── .github/workflows/
│   ├── ci.yml
│   ├── integration-macos.yml
│   ├── release.yml
│   └── gitleaks.yml             # existing
└── (install.sh / bootstrap.sh / lib/menu.sh / .stowrc — deleted at cutover)
```

### Layered design

Three layers, one event bus:

```
┌─────────────────────────────────────────────────────────┐
│  CLI (Cobra)         TUI (Bubble Tea)                   │
│  ─ install --all     ─ pick tools, see progress, logs   │
│  ─ install <tool>    ─ resolve conflicts inline         │
│  ─ status, update    ─ default when run with no args    │
└──────────────┬───────────────────────┬──────────────────┘
               │                       │
               └───────┬───────────────┘
                       ▼
              ┌────────────────────┐
              │  Runner            │
              │  ─ load registry   │
              │  ─ select tools    │
              │  ─ run steps       │
              │  ─ emit events     │
              │  ─ tolerate faults │
              └─────────┬──────────┘
                        │ events
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
   Step impls      Linker          Manifest registry
   (brew_*,        (native         (embedded + overlay)
    shell, etc.)   symlinks)
```

Principles:
- The runner owns control flow. Both CLI and TUI invoke the same runner; the only difference is the event sink.
- Steps are uniform: a `brew_cask` step and a `shell` step look identical to the runner and the TUI.
- Manifests are data. Adding a tool is editing YAML; no rebuild required if dropped in the overlay directory.
- No global state. Runner takes a `Context`, a list of selected tools, and an `EventSink`.

## Components

### Manifest schema

```yaml
# tools/ghostty.yaml
name: ghostty
description: Terminal emulator
platforms: [darwin]
tags: [gui, terminal]
depends_on: [homebrew]
steps:
  - type: brew_cask
    package: ghostty
configs:
  - source: ghostty             # relative to configs/ in the repo
    target: ~/.config/ghostty   # ~ expanded; created if missing
```

```yaml
# tools/oh-my-zsh.yaml — escape hatch
name: oh-my-zsh
description: Zsh framework with plugins
platforms: [darwin]
depends_on: [git]
steps:
  - type: shell
    name: install oh-my-zsh
    check: '[ -d "$HOME/.oh-my-zsh" ]'
    install: |
      sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended
  - type: shell
    name: install zsh-autosuggestions
    check: '[ -d "$HOME/.oh-my-zsh/custom/plugins/zsh-autosuggestions" ]'
    install: |
      git clone https://github.com/zsh-users/zsh-autosuggestions \
        "$HOME/.oh-my-zsh/custom/plugins/zsh-autosuggestions"
```

Required fields: `name`, `steps` (≥1). Optional: `description`, `platforms` (default: all), `tags` (default: empty), `depends_on` (default: empty), `configs` (default: empty).

### Step types (v1)

| Type | Action | Required fields |
|------|--------|----------------|
| `homebrew_bootstrap` | Install Homebrew if missing | — |
| `brew_package` | `brew install <pkg>` | `package` |
| `brew_cask` | `brew install --cask <pkg>` | `package` |
| `brewfile` | `brew bundle --file=<path>` | `path` |
| `shell` | Run shell snippet for `install` (and optional `update`); `check` short-circuits | at least one of `install`/`update`; `check` optional |
| `git_clone` | Clone-or-pull a repo | `url`, `dest` |
| `npm_global` | `npm install -g <pkg>` | `package` |
| `git_config` | Set git config | `key`, `value`, optional `scope: repo\|global` (default: repo) |

Every step implements:

```go
type Step interface {
    Type() string
    Check(ctx context.Context, env Env) (satisfied bool, err error)
    Run(ctx context.Context, env Env, sink EventSink) error
}
```

`Env` carries the `Exec` and `FS` seams (see Testing), current `Platform`, and `HomeDir`.

### Tool registry

- At startup, embed `tools/*.yaml` via `//go:embed`.
- Then read `$DOTFILES_DIR/tools/*.yaml` and `~/.config/dot/tools/*.yaml`. Same `name` → overlay file replaces the embedded one entirely (no field-level merge — keeps semantics simple).
- Validate after load: schema, unknown step types, unresolved `depends_on`, cycles. Validation failures are fatal at startup.
- `dot list` shows: name | source (embedded/overlay) | platforms | tags.

### Linker

- Walks `configs/<source>/` recursively.
- For each leaf: if target missing → symlink; if target is a symlink to source → no-op; if target exists and differs → conflict.
- Conflict actions: `Backup` (move to `~/.dotfiles_backup_<timestamp>/<original-path>`, then symlink), `Overwrite` (delete then symlink), `Skip` (leave existing), `Abort` (stop deploy for this tool).
- TUI: interactive modal per conflict, with "apply to remaining" option.
- CLI: `--on-conflict=backup|overwrite|skip|abort` flag; default `abort` if non-interactive.
- All conflicts for a tool are collected before any action is applied — either all symlinks for that tool succeed or none. Backups are atomic per file.

### Runner

```go
func Run(ctx context.Context, plan Plan, sink EventSink) Result
```

- `Plan` is an ordered list of tools (topologically sorted by `depends_on`).
- Tools run **sequentially**, not in parallel (homebrew serializes anyway; logs are readable; conflicts are interactive).
- Steps within a tool also run sequentially.
- Fault tolerance: a failed step fails its tool; tools downstream via `depends_on` are skipped with reason "dependency failed: X". Independent tools continue.
- End-of-run summary lists successes, skips, and failures.

### TUI (Bubble Tea)

Three screens, each a `tea.Model`:

1. **Picker** — list of tools (Bubbles `list`), space to toggle, `a` select all, enter to run. Filterable by tag.
2. **Runner** — split pane: top is per-tool status list (spinner / ✓ / ✗ / skip); bottom is rolling log pane for the focused tool. Tab cycles focus.
3. **Conflict prompt** — modal overlay during deploy. Choices: backup / overwrite / skip / abort. Optional "apply to remaining."

After a run, `r` retries failed tools, `l` opens the log pane for a finished tool, `q` quits.

### CLI surface (Cobra)

```
dot                          # opens TUI picker
dot install [<tool>...]      # install selected (or --all, or --tag); --on-conflict
dot update  [<tool>...]      # alias for install (reconcile)
dot status  [<tool>...]      # read-only Check() for each tool
dot deploy  [<tool>...]      # re-create symlinks only, skip install steps
dot list                     # list known tools
dot version
```

Global flags: `--non-interactive`, `--verbose`, `--config-dir`, `--dotfiles-dir`.

### Locating the dotfiles directory

The binary is installed standalone (e.g., `~/.local/bin/dot`) but needs a dotfiles repo to resolve `configs/` sources and to read overlay manifests from `tools/`. Resolution order, first hit wins:

1. `--dotfiles-dir` flag if set
2. `DOTFILES_DIR` env var if set
3. Current working directory if it contains `configs/` (so running `dot install` from a cloned repo Just Works)
4. `~/dotfiles` if it exists and contains `configs/`
5. `~/.dotfiles` if it exists and contains `configs/`

If none resolve and the command needs `configs/` (`install` for tools with `configs:`, or `deploy`), exit 2 with a clear message pointing at the env var and flag. Commands that don't need it (`list`, `version`, and `status` for tools without `configs:`) work fine without a dotfiles directory.

The `--config-dir` flag points at the user overlay directory for manifests (defaults to `~/.config/dot`); these are independent of the dotfiles repo.

## Data flow

### Startup (every command)

1. Cobra parses args → command + flags.
2. `platform.Detect()` → `{OS: darwin, Arch: arm64}`.
3. `manifest.LoadRegistry()` reads embedded manifests, overlays user manifests, validates, filters by current platform.
4. `tool.Select(args, flags)` resolves the target set (interactive picker, `--all`, `--tag`, explicit names).
5. `tool.ExpandDeps(selected)` transitively pulls in `depends_on` targets that aren't already selected. The TUI shows the auto-included tools and asks for confirmation before running; the CLI prints "including transitive dependencies: X, Y" and proceeds. `--no-deps` skips expansion and fails fast if a required dependency isn't installed.
6. `tool.Sort(expanded)` topologically orders by `depends_on`.
7. The verb (`install` / `update` / `status` / `deploy`) executes.

### Install/update flow

```
for each tool in plan:
  emit ToolStarted
  for each step in tool:
    emit StepStarted
    if step.Check() satisfied → emit StepSkipped
    else
      step.Run(ctx, sink)
        └─ exec.Cmd, stdout/stderr piped line-by-line
           each line → emit LogLine
      if err: emit StepFailed → break tool loop
      else:   emit StepFinished
  if any step failed → emit ToolFailed
  else if tool.configs declared → linker.Link(configs, sink)
  emit ToolFinished

for tools depending on a failed tool: emit ToolSkipped
```

### Event model

```go
type Event struct {
    Kind     EventKind   // ToolStarted, StepStarted, LogLine, StepSkipped,
                         // StepFinished, StepFailed, ToolFinished, ToolFailed,
                         // ToolSkipped, ConflictPrompt, ConflictResolved
    Tool     string
    Step     string
    Line     string      // for LogLine
    Level    Level       // info, warn, error
    Err      error       // for failures
    Conflict *Conflict   // for ConflictPrompt; carries the resolver channel
}

type EventSink interface { Send(Event) }
```

Two implementations:
- `StreamSink` — formats events to stdout/stderr (CLI, non-TTY, or `--non-interactive`).
- `TUISink` — wraps a Bubble Tea program; each event becomes a `tea.Msg`.

### Conflict resolution flow

The `Conflict` event carries a channel back to the linker. The TUI puts up a modal; the user picks; the channel returns the choice; the linker proceeds. The CLI sink resolves it immediately from `--on-conflict` without ever blocking.

### Status flow

Read-only. For each tool, every step's `Check()` is called. Per-tool status: `installed` | `missing` | `partial`. No mutations. Network-touching checks (e.g., `brew outdated`) are opt-in via `--deep`.

## Error handling

### Categories and policy

| Category | Examples | Policy |
|----------|----------|--------|
| Manifest validation | Unknown step type, missing `depends_on` target, cycle, malformed YAML | Fatal at startup. Print all errors. Exit 2. |
| Selection error | Unknown tool name, no matches for `--tag` | Fatal pre-flight. Suggest similar names. Exit 2. |
| Step subprocess failure | `brew install` non-zero, `curl` fails, `git clone` 404 | Fail the tool, capture exit code + stderr tail, emit `ToolFailed`. Runner moves on. |
| Dependency failure | Tool B `depends_on: [A]`, A failed | B skipped with reason. `ToolSkipped`. Not counted as failure. |
| Platform mismatch | Tool declares `platforms: [linux]` on darwin | Filtered out at load. If explicitly requested, fail pre-flight. |
| Conflict (linker) | Existing real file at target | Resolved via prompt or `--on-conflict`. If `abort`, fail the tool. |
| Permission denied | `~/.config` write denied | Same as subprocess failure; surface message; continue. |
| Interrupt (Ctrl+C) | User cancels | Cancel `context.Context`. Step's subprocess gets SIGTERM, then SIGKILL after 5s. Remaining tools `Cancelled`. Exit 130. |
| Internal error / panic | Bug in our code | Recover panic in runner; treat as `StepFailed` with stack trace in error. |

### Exit codes

```
0   success (zero failures; skips are not failures)
1   one or more tools failed
2   pre-flight error (bad manifest, bad CLI args, unknown tool)
130 cancelled (Ctrl+C)
```

### Fault tolerance principles

1. One failure does not abort the run. Failed tools fail themselves and their dependents; independents continue.
2. Failures are loud but local. Full subprocess output goes to a per-tool log buffer; last ~50 lines of stderr surface in the end-of-run summary; full log always accessible in TUI.
3. Idempotency is a contract. Every step's `Run()` must be safe to re-run. If it can't (rare), it must implement `Check()` properly so it short-circuits when satisfied.
4. No half-states for symlinks. All conflicts for a tool are collected before any action is applied. Backups are atomic per file.

### End-of-run summary (TUI)

```
✓ homebrew         (Bootstrap + brew bundle, 12.3s)
✓ git              (configs deployed: ~/.gitconfig, ~/.gitignore_global)
✗ oh-my-zsh        (step 'install zsh-autosuggestions' failed: exit 128)
   ╰─ git clone https://github.com/zsh-users/zsh-autosuggestions ...
      fatal: destination path already exists and is not empty
~ claude-code      skipped (depends_on: nvm — nvm failed)
✗ nvm              (step 'install nvm' failed: curl: (6) Could not resolve host)
✓ vim
✓ k9s

────────────────────────────────────────────────────────
5 succeeded, 1 skipped, 2 failed.  Total: 38.1s
Press 'l' to view logs, 'r' to retry failed, 'q' to quit.
```

CLI (non-interactive) shows the same data in plain text. Exit 1.

### Retry

- TUI: `r` after a run re-invokes the runner with just the failed tools.
- CLI: user re-runs `dot install <failed-tool>` themselves. No automatic retry.

### Logging

- Per-run log file: `~/.local/state/dot/runs/<timestamp>.log` — full event stream + subprocess output. Keep last 10 runs.
- TUI log pane streams active tool's log; `l` on a finished tool opens its log.
- `--verbose` mirrors everything to stderr in CLI mode.

## Testing and tooling

### Testability seams

```go
// All subprocess execution flows through this.
type Exec interface {
    Run(ctx context.Context, cmd string, args []string, env []string, sink LineSink) error
}
// Real impl wraps os/exec; fake impl returns scripted output per matched command.

// All filesystem operations flow through this.
type FS interface {
    Stat(path string) (FileInfo, error)
    Symlink(source, target string) error
    Readlink(path string) (string, error)
    Mkdir(path string, perm os.FileMode) error
    Rename(old, new string) error
    Remove(path string) error
    Walk(root string, fn WalkFn) error
}
// Real impl passes through os/filepath; fake impl is an in-memory tree.
```

Steps never reach for `os.Exec` or `os.Symlink` directly — they go through `Env.Exec` / `Env.FS`.

### Unit tests

| Component | What's tested | How |
|-----------|---------------|-----|
| `manifest` | YAML parsing, validation errors, schema, overlay merge precedence | Table-driven tests with embedded YAML strings |
| `tool` | Topological sort, cycle detection, dependency-failure propagation, platform filtering | Pure functions |
| `step` | Builtin step `Check()` and `Run()`: command construction, output parsing, idempotency | Fake `Exec` with scripted responses per command match |
| `linker` | Symlink/no-op/conflict decision, conflict actions, backup paths, walking | Fake `FS` in-memory tree |
| `runner` | Tool ordering, dependency skip propagation, event emission, panic recovery, context cancellation | Fake step impls + capturing `EventSink` |
| `platform` | OS detection branches | Mockable `runtime` |

### TUI tests

Use [`teatest`](https://github.com/charmbracelet/x/tree/main/exp/teatest) (charmbracelet's official Bubble Tea test harness).

- Feed key events; assert on rendered output (string match or snapshot).
- Cover: picker toggles, runner pane updates on events, conflict prompt resolution, retry shortcut.
- Snapshot files committed under `internal/tui/testdata/`.

### Integration tests

- Build tag `integration` (excluded from default `go test ./...`).
- Trivial tools (`git_config`, `shell` with no-op install) run against a temp `$HOME`.
- `dot deploy` exercises real symlinks + conflict resolution via `--on-conflict=backup`.
- Heavy tools (brew, npm, oh-my-zsh) are not run in default CI. A separate workflow `integration-macos.yml` runs them on `macos-latest` via manual trigger or nightly.

### Repo tooling

`Makefile`:

```
make build         go build -o bin/dot ./cmd/dot
make test          go test ./...
make test-int      go test -tags=integration ./...
make lint          golangci-lint run
make fmt           gofumpt -w . && goimports -w .
make fmt-check     gofumpt -l . && goimports -l .    # exits non-zero if any diff
make tidy          go mod tidy
make snapshot      go test -tags=teatest -update ./internal/tui/...
make release       goreleaser release --clean
make release-dry   goreleaser release --snapshot --clean
```

Linting via `golangci-lint` with:
- `govet`, `staticcheck`, `errcheck`, `ineffassign`, `unused` — correctness
- `revive` — style
- `gofumpt`, `goimports` — formatting (also enforced via `make fmt-check`)
- `gocyclo`, `funlen` — complexity caps
- `gosec` — basic security
- `errorlint` — error wrapping discipline

Formatting: `gofumpt` + `goimports`, enforced in CI.

### GitHub Actions

```
.github/workflows/
├── ci.yml                  # PR + push to main: build, test, lint, fmt-check
├── integration-macos.yml   # workflow_dispatch + nightly on macos-latest
├── release.yml             # tag push v* → goreleaser publishes to GitHub Releases
└── gitleaks.yml            # existing
```

`ci.yml`:
- Matrix `{darwin, linux} × {amd64, arm64}` for build only (cross-compile validation).
- Unit tests on `ubuntu-latest`.
- `golangci-lint` + `make fmt-check`.

`release.yml`:
- `goreleaser/goreleaser-action` produces tarballs per platform, SHA256 checksums, changelog.
- README links to a small `install.sh` bootstrap script (committed in repo) that detects platform, downloads the right binary, drops it in `~/.local/bin/dot`.

### Pre-commit (extend `.githooks/pre-commit`)

- Keep `gitleaks`.
- Add `gofumpt -l` check (fail if Go files would change).
- Add `golangci-lint run --new-from-rev=HEAD~ --fast` (only lint what changed).

### Migration tests

Each tool currently installed by `install.sh` (homebrew-tools, rust, oh-my-zsh, nvm, claude-code, git-hooks) must be expressible as a manifest. A test in `make test` asserts the registry contains all of them after the rewrite. The `configs/<tool>/` layout is honored unchanged.

## Out of scope (v1)

- Full Linux backend parity (apt/dnf step types). Manifests can declare `platforms:`; Linux backends added when actually needed.
- Homebrew tap distribution. Goreleaser pipeline is designed so this is one config block away when desired.
- Version pinning for tools. `update` is reconciliation, not version negotiation.
- Parallel tool installs.
- macOS GUI defaults automation.
- SSH config management.
