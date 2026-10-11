# Plan: kind, Codex CLI, and jq/yq rules for CLAUDE.md

Branch: `feat/kind-codex-json-yaml-rules`

## Scope

1. **kind**: local Kubernetes clusters in Docker.
2. **Codex**: OpenAI's Codex CLI (`codex`), installed as the Homebrew cask `codex`.
3. **Default `CLAUDE.md`**: tell agents to use `yq` to parse, edit, and validate YAML,
   use `jq` for JSON, and never write custom Python scripts for either.

Out of scope: the Codex desktop app (`codex-app` cask), Codex configuration
(`~/.codex/`), Codex authentication, Linux support.

## Findings

### How tools are installed

- macOS only. `install.sh` exits if `$OSTYPE` is not `darwin*`.
- `Brewfile` is the single source of truth for Homebrew formulae and casks.
  `install.sh` installs it with `brew bundle --file=Brewfile` under the
  `homebrew-tools` item, so adding a tool takes one line.
- Tools that aren't in Homebrew (rust, oh-my-zsh, nvm, claude-code) each get an
  `add_item` entry and an `if confirm_item ...; then ... fi` block in `install.sh`.
- `lib/menu.sh` handles prompting: `[Y/n]`, or `--all` to skip prompts. It must
  stay Bash 3.2 compatible.
- Versions aren't pinned. Homebrew installs whatever is latest, nvm installs the
  latest LTS, and `mise` (`configs/mise/.config/mise/config.toml`) manages
  per-project tool versions. No change is needed here.
- The `homebrew-tools` description in `install.sh` hard-codes the count:
  `"22 packages via Brewfile"`. That's 18 `brew` lines plus 4 `cask` lines
  (the `tap` line isn't counted).

### kind

- **Already installed.** `Brewfile` has `brew "kind"` (added in #9), and
  `AGENTS.md` lists it in the install.sh package list.
- What's missing: `README.md` doesn't mention kind. Line 70 lists only
  `k9s, kubectl (Kubernetes)`. kubectx, helm, ko, oras, and kluctl from #9 are
  also missing there, but they're out of scope (see Open questions).
- kind needs a running Docker daemon. Docker Desktop is already a cask in the
  Brewfile.

### Codex

- The Homebrew cask `codex` (homepage https://github.com/openai/codex) is
  binary-only. It puts a `codex` binary in the Homebrew bin dir and generates
  bash/zsh/fish completions. It doesn't install anything into `/Applications`.
- The cask (0.160.0 when checked) is slightly behind npm `@openai/codex`
  (0.162.1). We accept the lag in exchange for keeping it in the Brewfile and
  letting `brew bundle` handle idempotency.
- The user still has to authenticate (`codex login` or `OPENAI_API_KEY`). No
  credentials or Codex config go into the repo.

### CLAUDE.md template

- The template is `configs/claude/.claude/CLAUDE.sample.md` (tracked).
- `bootstrap.sh` stows `configs/claude`, which symlinks `~/.claude` to
  `configs/claude/.claude`. It then processes every `*.sample.*` file. For
  `CLAUDE.sample.md` it **copies to `CLAUDE.md` only if that file is missing**.
  Only `.json` samples are deep-merged with `jq`, and Markdown is never merged.
  `README.md` (the "`.sample` template" table) documents this.
- So updating the template **won't change an existing `~/.claude/CLAUDE.md`**.
  On this machine, `configs/claude/.claude/CLAUDE.md` (untracked, identical to
  the current template) has to be edited by hand or deleted and regenerated.
- The `# Tools Usage` section already has a narrower rule:
  `- For YAML file validation, always use \`yq\`. Never write a Python script for verification.`
  The new rules replace it.
- `jq` and `yq` are already in the `Brewfile`, so there are no new dependencies.

## Affected files

| File | Change |
|------|--------|
| `Brewfile` | Add `cask "codex"` |
| `install.sh` | Count `22` → `23` in the `homebrew-tools` description |
| `configs/claude/.claude/CLAUDE.sample.md` | Replace the YAML rule with jq/yq rules |
| `README.md` | Mention kind and Codex in the install lists; mention the jq/yq rule in the CLAUDE.md section |
| `AGENTS.md` | Mention Codex in the casks list and install table |

Not touched: `bootstrap.sh`, `lib/menu.sh`, untracked runtime files under
`configs/claude/.claude/`, and the existing uncommitted changes to
`configs/git/.gitconfig`, `configs/git/.gitignore_global`, `configs/zsh/.zshrc`.

## Proposed changes

### 1. kind (docs only)

`README.md`, "Homebrew packages and casks" list:

```diff
-- k9s, kubectl (Kubernetes)
+- k9s, kubectl, kind (Kubernetes)
```

### 2. Codex CLI (Brewfile cask)

`Brewfile`, in the cask section after `cask "cursor"`:

```ruby
cask "codex"
```

`install.sh`:

```diff
-add_item "homebrew-tools" "22 packages via Brewfile"
+add_item "homebrew-tools" "23 packages via Brewfile"
```

`README.md`, "Homebrew packages and casks" list, after Cursor:

```markdown
- Codex (OpenAI coding agent CLI)
```

`AGENTS.md`:

- Line 59 casks list: add `codex (OpenAI coding agent CLI)`.
- "Tool Installation Details" table: add a row
  `| Codex CLI | \`brew install --cask\` | Homebrew bin (binary-only cask) |`.
  Don't add it to the `Ghostty, Raycast, Cursor, Docker` → `/Applications/` row.

### 3. jq/yq rules in the CLAUDE.md template

`configs/claude/.claude/CLAUDE.sample.md`, `# Tools Usage` section. Replace:

```markdown
- For YAML file validation, always use `yq`. Never write a Python script for verification.
```

with:

```markdown
- Always use `yq` to parse, query, edit, and validate YAML files.
- Always use `jq` to parse, query, edit, and validate JSON files.
- Never write custom Python scripts (including `python -c` one-liners) to process, edit, or validate YAML or JSON.
```

The draft-PR rule after it stays as is.

`README.md`, "`CLAUDE.md` — global behavioral rules" list, after the GitHub bullet:

```markdown
- **Structured data**: use `yq` for YAML and `jq` for JSON, never ad-hoc Python scripts.
```

### Rollout to existing machines

Since bootstrap keeps an existing `~/.claude/CLAUDE.md`, apply the template
change on this machine by making the same edit to `~/.claude/CLAUDE.md`, or by
deleting it and re-running `./bootstrap.sh` (answer yes only for `claude`).
Changing bootstrap's copy-if-missing behaviour is out of scope.

## Implementation order

1. `Document kind in README`: README line only.
2. `Add Codex CLI cask`: `Brewfile`, `install.sh` count, `README.md`, `AGENTS.md`.
3. `Require yq/jq for YAML and JSON in default CLAUDE.md`:
   `CLAUDE.sample.md` and `README.md`.
4. Make the same edit to the local `~/.claude/CLAUDE.md` (not committed).

Stage files by explicit path in each commit so the unrelated uncommitted
changes and untracked runtime files stay out.

## Validation

```bash
# Scripts still parse
bash -n install.sh
bash -n bootstrap.sh

# Brewfile parses and lists the new cask; count matches install.sh
brew bundle list --file=Brewfile --all
brew bundle list --file=Brewfile --cask | grep -x codex
[ "$(grep -cE '^(brew|cask) ' Brewfile)" = 23 ] && grep -q '"23 packages via Brewfile"' install.sh

# kind and Codex installed and runnable
brew bundle check --file=Brewfile --verbose   # after ./install.sh (homebrew-tools only)
kind version
codex --version

# Template contains the new rules and not the old one
grep -n -E '`yq`|`jq`|Python' configs/claude/.claude/CLAUDE.sample.md
! grep -q 'For YAML file validation' configs/claude/.claude/CLAUDE.sample.md

# Template is applied for a fresh install (missing target → copied)
diff configs/claude/.claude/CLAUDE.sample.md ~/.claude/CLAUDE.md

# Stow dry run for the claude package reports no conflicts
stow -n -v claude

# No secrets in staged changes (also runs via pre-commit hook)
gitleaks protect --staged --redact --verbose

# Docs mention the new tools
grep -n -E 'kind|Codex' README.md AGENTS.md
```

Manual check: start a new Claude Code session and ask it to change a value in a
YAML file and a JSON file. It should use `yq -i` / `jq` and not write a Python script.

## Open questions

- `README.md` also omits kubectx, helm, ko, oras, and kluctl, which were added
  in #9. Should we fix that here or in a separate change? This plan leaves it
  out to keep the diff scoped.
- Should Codex get its own `AGENTS.md`-style global instructions
  (`~/.codex/AGENTS.md`) with the same jq/yq rule? This plan doesn't include it.
