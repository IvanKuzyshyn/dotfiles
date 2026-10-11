# Plan: kind + Codex installation, yq/jq rules in CLAUDE.md template

Status: research only. Nothing here is implemented yet.

## Goals

1. Install `kind` and the OpenAI Codex CLI using the conventions the repo already follows.
2. Extend the Claude `CLAUDE.md` template so agents use `yq` for YAML and `jq` for JSON
   (parse, edit, validate), and never write custom Python scripts for these operations.

## Findings

### How tools are installed

- macOS only. `install.sh` exits on non-Darwin `$OSTYPE`.
- `Brewfile` is the single source of truth for Homebrew formulae and casks.
  `install.sh` runs `brew bundle --file="$SCRIPT_DIR/Brewfile"` under the
  `homebrew-tools` item. `brew bundle` is idempotent.
- Tools that aren't installed through brew (rust, oh-my-zsh, nvm, claude-code, git-hooks) get
  a dedicated `add_item` plus a `confirm_item` block in `install.sh`.
- The `homebrew-tools` description in `install.sh` hardcodes a count:
  `"22 packages via Brewfile"`. That is 18 `brew` entries plus 4 `cask` entries. Taps aren't counted.
- Versions aren't pinned. Brew installs the latest version. `mise` handles per-project
  tool versions through `mise.toml`, but it isn't used for globally installed tools.
- Docs that list installed tools: `README.md` ("Installation → Homebrew packages
  and casks") and `AGENTS.md` ("Setup Workflow" and the "Tool Installation Details" table).

### kind

- `kind` is **already** in `Brewfile` (`brew "kind"`), added in 76e71ed, and
  `AGENTS.md` already lists it under the installed packages.
- The only gap is `README.md`, where the Kubernetes line reads `k9s, kubectl (Kubernetes)`.
  It also leaves out kubectx, helm, ko, oras, and kluctl. This plan only adds kind, to keep scope tight.
- Runtime dependency: kind needs a container runtime. The `docker` cask (Docker Desktop)
  is already in `Brewfile`, so no new dependency is required.
- Locally installed at `/opt/homebrew/bin/kind` (Homebrew stable 0.33.0).

### Codex

- Product: OpenAI Codex CLI (`codex`, https://github.com/openai/codex).
- Chosen method: **Homebrew cask** `codex`, which installs only a binary (`bin/codex`) and
  generates zsh completions. This keeps `Brewfile` as the single source of truth.
  The other two options were npm `@openai/codex` and OpenAI's standalone installer.
  Neither fits as well as a single Brewfile line.
- The current machine already has Codex from the standalone installer:
  `~/.local/bin/codex -> ~/.codex/packages/standalone/current/bin/codex` (0.162.1).
  After `brew bundle`, two `codex` binaries will exist, and whichever comes first on `PATH` wins.
  This only matters on this machine. A fresh machine gets just the brew copy. If you want
  a single copy, remove the standalone install by hand. That step is outside the repo.
- Codex doesn't add a config package here (`~/.codex/` isn't managed by stow).
  This plan doesn't add one.

### Claude CLAUDE.md template

- Source template: `configs/claude/.claude/CLAUDE.sample.md`.
- It is applied by `bootstrap.sh`. Stow symlinks the `claude` package, then the `*.sample.*` loop
  copies `CLAUDE.sample.md` to `~/.claude/CLAUDE.md` **only if the target is missing**.
  Only `*.json` samples get merged. Markdown samples are left alone once they exist.
- On this machine, `~/.claude` is itself a symlink to `configs/claude/.claude` (stow
  folded the whole directory). So the deployed `CLAUDE.md` is the untracked file
  `configs/claude/.claude/CLAUDE.md`, which is currently identical to the sample.
  Re-running `./bootstrap.sh` **won't** pick up template changes. The deployed copy
  has to be updated by hand (see Validation).
- The template's existing `# Tools Usage` section already has a YAML-only rule:
  `For YAML file validation, always use yq. Never write a Python script for verification.`
  This plan replaces that rule with broader rules covering YAML and JSON instead of
  adding a near-duplicate.
- Dependencies: `jq` and `yq` are both already in `Brewfile`. `bootstrap.sh` already
  relies on `jq` for the settings merge.
- Related, but out of scope: `settings.sample.json` allows `Bash(jq:*)` but not `yq`,
  so `yq` calls will still prompt for permission. Add `Bash(yq:*)` only if requested.
- `README.md` summarizes the template's sections in "`CLAUDE.md` — global behavioral
  rules". That summary should mention the new rule.

## Affected files

| File | Change |
|------|--------|
| `Brewfile` | Add `cask "codex"` to the cask block |
| `install.sh` | Change the `homebrew-tools` description from `22` to `23 packages via Brewfile` |
| `configs/claude/.claude/CLAUDE.sample.md` | Replace the YAML-only rule in `# Tools Usage` with yq/jq rules |
| `README.md` | Add kind to the Kubernetes line, add Codex to the casks list, add a structured-data bullet to the CLAUDE.md summary |
| `AGENTS.md` | Add codex to the "Installs casks" line and the Tool Installation Details table |

No changes to `bootstrap.sh`, `lib/menu.sh`, or `.stowrc`.

## Proposed changes

### Brewfile

```ruby
cask "ghostty"
cask "raycast"
cask "cursor"
cask "codex"
cask "docker"
```

### install.sh

```bash
add_item "homebrew-tools" "23 packages via Brewfile"
```

### configs/claude/.claude/CLAUDE.sample.md (`# Tools Usage`)

Replace:

```markdown
- For YAML file validation, always use `yq`. Never write a Python script for verification.
```

with:

```markdown
- Use `yq` to parse, query, edit, and validate YAML files.
- Use `jq` to parse, query, edit, and validate JSON files.
- Never write custom Python scripts (including `python -c` one-liners) to process or validate YAML or JSON.
```

Keep the existing draft-PR rule unchanged.

### README.md

- `- k9s, kubectl (Kubernetes)` → `- k9s, kubectl, kind (Kubernetes)`
- Add `- Codex (OpenAI coding agent CLI)` to the casks list, after Cursor.
- In the CLAUDE.md summary, add:
  `- **Structured data** — use \`yq\` for YAML and \`jq\` for JSON, never ad-hoc Python scripts.`

### AGENTS.md

- "Installs casks" line: add `codex (OpenAI coding agent CLI)`.
- Tool Installation Details table: add the row `| Codex CLI | \`brew install --cask\` | Homebrew bin (binary-only cask) |`.

## Implementation order

1. Make the `Brewfile` and `install.sh` changes in one commit, so the count stays consistent with the Brewfile.
2. `CLAUDE.sample.md` rule change (commit).
3. `README.md` and `AGENTS.md` doc updates (commit).
4. Run validation, then sync the deployed `~/.claude/CLAUDE.md` by hand.

## Validation

```bash
# Script syntax
bash -n install.sh
bash -n bootstrap.sh

# Brewfile parses and lists the new cask; count matches install.sh (expect 23)
brew bundle list --file=Brewfile --all
brew bundle list --file=Brewfile --brews --casks | wc -l
grep -n 'packages via Brewfile' install.sh

# Install and verify
brew bundle --file=Brewfile
brew bundle check --file=Brewfile --verbose
kind version
codex --version
which -a codex          # shows brew vs standalone copy on this machine

# kind works end-to-end (requires Docker Desktop running)
kind create cluster --name plan-smoke && kubectl cluster-info --context kind-plan-smoke
kind delete cluster --name plan-smoke

# Template content
grep -n -A4 '^# Tools Usage' configs/claude/.claude/CLAUDE.sample.md
diff configs/claude/.claude/CLAUDE.sample.md ~/.claude/CLAUDE.md   # shows the deployed copy is stale

# Sync the deployed CLAUDE.md (bootstrap won't overwrite an existing file).
# Review the diff above first: this overwrites any local edits.
cp configs/claude/.claude/CLAUDE.sample.md ~/.claude/CLAUDE.md

# Stow dry run for the claude package is unchanged
stow -n -v claude

# Secret scan on staged changes
gitleaks protect --staged --redact --verbose
```

## Open items

- Should the standalone Codex install on this machine (`~/.codex/packages/standalone`)
  be removed after switching to the cask?
- Should `Bash(yq:*)` be added to `settings.sample.json` permissions so yq usage
  doesn't prompt? This wasn't part of the request, so it's not included.
- Prior branches cover overlapping work: `feat/kind-codex-json-yaml-rules`, local and on origin,
  already contains implementation commits. Plan-only branches also exist:
  `add-kind-pi-and-json-yaml-rules` and `feat/kind-pi-json-yaml-rules`. Decide which
  branch to carry forward and delete the rest.
