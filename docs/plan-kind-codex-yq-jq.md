# Plan: kind + Codex installation, yq/jq rules in the CLAUDE.md template

Status: research only. Nothing in this document is implemented yet.

## Goals

1. Install `kind` and the OpenAI Codex CLI using the repository's existing conventions.
2. Extend the Claude `CLAUDE.md` template so agents use `yq` to parse, edit, and validate
   YAML and `jq` for JSON, and never write custom Python scripts for these operations.

## Decisions

- **Codex product:** OpenAI Codex CLI (`codex`, https://github.com/openai/codex).
- **Codex install method:** Homebrew cask `cask "codex"` in `Brewfile` (confirmed with the user).
  The npm package (`@openai/codex`), OpenAI's standalone installer, and the desktop app
  (`cask "codex-app"`) were considered and rejected.

## Findings

### How tools are installed

- macOS only. `install.sh` exits on non-Darwin platforms.
- `Brewfile` is the single source of truth for Homebrew formulae, taps, and casks.
  `install.sh` runs `brew bundle --file="$SCRIPT_DIR/Brewfile"` under the `homebrew-tools` item.
- Non-brew tools (rust, oh-my-zsh, nvm, claude-code, git-hooks) each get an `add_item`
  registration plus a `confirm_item` block in `install.sh`. Prompts come from `lib/menu.sh`.
- The `homebrew-tools` item description hard-codes the package count:
  `"22 packages via Brewfile"` (18 `brew` + 4 `cask` lines today).
- Versions are not pinned. Brew installs the latest version. `mise` handles per-project
  tool versions and isn't involved here.

### kind

- **Already installed.** `brew "kind"` has been in `Brewfile` since PR #9 (`add-k8s-tools`).
  Installed locally: `kind v0.33.0`. Brew ships `_kind` zsh completion into
  `/opt/homebrew/share/zsh/site-functions`.
- `AGENTS.md` already lists kind in the installed packages.
- **Gap:** `README.md` doesn't mention kind. The Kubernetes line reads `k9s, kubectl (Kubernetes)`.
  It also omits kubectx, helm, ko, oras, and kluctl, which came in the same PR.
- Runtime dependency: kind needs a container runtime. `cask "docker"` (Docker Desktop) is
  already in `Brewfile`, so no new dependency is needed.
- Required work: documentation only.

### Codex

- `brew info --cask codex` reports version 0.162.1. The cask installs only `bin/codex` and
  generates bash/zsh/fish completions. It doesn't depend on Node.js, so it doesn't need nvm,
  unlike the `claude-code` item.
- This machine already has Codex from OpenAI's standalone installer at `~/.local/bin/codex`
  (0.162.1). After `brew bundle` there will be two `codex` binaries, and the one earlier on
  `PATH` wins. This only affects this machine; a fresh machine gets only the brew copy.
  Removing the standalone copy is a manual step outside the repo.
- `~/.codex/` (Codex config) isn't managed by stow. This plan doesn't add a `configs/codex`
  package.

### CLAUDE.md template

- Source template: `configs/claude/.claude/CLAUDE.sample.md` (tracked).
- How it's applied: `bootstrap.sh` stows `configs/claude`, then handles every `*.sample.*`
  file. For a `.md` sample, it copies to `~/.claude/CLAUDE.md` **only if the target
  doesn't exist**. Unlike `settings.json`, it never merges, so an existing `CLAUDE.md` is
  never updated.
- On this machine, `~/.claude` is a folded stow symlink to `configs/claude/.claude`.
  The live `CLAUDE.md` therefore sits in the repo as an untracked file. It's currently
  byte-identical to the sample.
- The `# Tools Usage` section already has a narrower rule:
  `For YAML file validation, always use yq. Never write a Python script for verification.`
  It should be replaced with broader rules, not duplicated.
- `README.md` → "`CLAUDE.md` — global behavioral rules" summarizes the template by
  section. It doesn't mention Tools Usage today.
- Dependencies: `jq` and `yq` are already in `Brewfile` (installed: `jq-1.8.2`,
  mikefarah `yq v4.53.6`). No new dependencies.

## Affected files

| File | Change |
|------|--------|
| `Brewfile` | Add `cask "codex"` to the cask block |
| `install.sh` | Change the `homebrew-tools` description to `"23 packages via Brewfile"` |
| `configs/claude/.claude/CLAUDE.sample.md` | Replace the YAML-only rule in `# Tools Usage` with the yq/jq rules |
| `README.md` | Add kind to the Kubernetes bullet, add Codex to the brew list, and add a Tools Usage bullet to the CLAUDE.md summary |
| `AGENTS.md` | Add codex to the casks list and the Tool Installation Details table |

No changes to `bootstrap.sh`, `lib/menu.sh`, `.stowrc`, or `settings.sample.json`.

## Proposed changes

### `Brewfile`

```ruby
cask "ghostty"
cask "raycast"
cask "cursor"
cask "codex"
cask "docker"
```

### `install.sh`

```bash
add_item "homebrew-tools" "23 packages via Brewfile"
```

### `configs/claude/.claude/CLAUDE.sample.md`

Replace:

```markdown
- For YAML file validation, always use `yq`. Never write a Python script for verification.
```

with:

```markdown
- For YAML files, always use `yq` to parse, edit, and validate.
- For JSON files, always use `jq` to parse, edit, and validate.
- Never write custom Python scripts to parse, edit, or validate YAML or JSON.
```

### `README.md`

- `- k9s, kubectl (Kubernetes)` → `- k9s, kubectl, kind (Kubernetes)`.
  The other missing k8s tools (kubectx, helm, ko, oras, kluctl) are out of scope.
  Listing them is a separate docs fix.
- Add `- Codex (OpenAI coding agent CLI)` after `- Cursor (AI code editor)`.
- In the CLAUDE.md summary, add:
  `- **Tools usage** — `yq` for YAML and `jq` for JSON (parse, edit, validate), never
  ad-hoc Python scripts; draft PRs by default.`

### `AGENTS.md`

- "Installs casks:" line → add `codex (OpenAI coding agent CLI)`.
- Tool Installation Details table: change the casks row to
  `Ghostty, Raycast, Cursor, Codex, Docker`. Codex installs to `$(brew --prefix)/bin/codex`,
  not `/Applications/`, so add a separate row: `| Codex | brew install --cask codex | $(brew --prefix)/bin/codex |`.

### Applying the template to existing machines (manual, not a repo change)

`bootstrap.sh` won't overwrite an existing `~/.claude/CLAUDE.md`. After the template
changes, update the live file by hand. On this machine it's identical to the sample, so
a copy is safe:

```bash
diff ~/.claude/CLAUDE.md configs/claude/.claude/CLAUDE.sample.md   # confirm only the new rules differ
cp configs/claude/.claude/CLAUDE.sample.md ~/.claude/CLAUDE.md
```

On machines where the live file has diverged, merge the three lines by hand.

## Implementation order

1. `Brewfile`: add `cask "codex"`. Then update the count in `install.sh`.
2. `CLAUDE.sample.md`: replace the Tools Usage rule.
3. `README.md` and `AGENTS.md`: update the docs.
4. Run validation (below). Commit each logical change separately
   (Codex install, CLAUDE.md rules, docs).
5. Sync the live `~/.claude/CLAUDE.md` manually.

## Validation

```bash
# Script syntax
bash -n install.sh bootstrap.sh lib/menu.sh

# Brewfile parses and resolves; lists what's missing without installing
brew bundle check --file=Brewfile --verbose
brew bundle list --file=Brewfile --all | grep -E '^(kind|codex)$'

# Package count in install.sh matches Brewfile
grep -cE '^(brew|cask) ' Brewfile    # expect 23

# Install and smoke test (interactive: answer Y to homebrew-tools only)
./install.sh
kind version
codex --version
command -v -a codex                   # shows both copies on this machine

# kind end-to-end (Docker Desktop must be running)
kind create cluster --name smoke && kubectl cluster-info --context kind-smoke && kind delete cluster --name smoke

# Template content
grep -nE '`(yq|jq)`' configs/claude/.claude/CLAUDE.sample.md
grep -c 'Python' configs/claude/.claude/CLAUDE.sample.md   # expect 1

# Stow still applies cleanly (dry run)
stow -n -v claude

# Secret scan before commit (also enforced by the pre-commit hook)
gitleaks protect --staged --redact --verbose
```

## Out of scope / open notes

- `yq` isn't in the `permissions.allow` list in `settings.sample.json` (`jq` is), so Claude
  prompts before running `yq`. Adding `Bash(yq:*)` would support the new rule, but it wasn't
  requested. Decide separately.
- Missing k8s tools in `README.md` (kubectx, helm, ko, oras, kluctl).
- Because `~/.claude` is a folded stow symlink, Claude runtime files (`projects/`,
  `sessions/`, `backups/`, and others) show up as untracked files in the repo.
  Possible fixes: a `.gitignore` entry or a `--no-folding` stow setting.
- Codex configuration (`~/.codex/`) as a stow package.
