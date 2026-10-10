# Plan: kind, pi, and JSON/YAML tool rules

Branch: `feat/kind-pi-json-yaml-rules`

## Scope

1. **kind** — local Kubernetes clusters in Docker.
2. **pi** — multi-provider coding agent harness (`@mariozechner/pi-coding-agent`, binary `pi`).
3. **Default `CLAUDE.md`** — make agents always use `jq` for JSON and `yq` for YAML, and never Python scripts.

## Findings

- `kind` is **already installed**: `Brewfile` has `brew "kind"` (added in #9), and
  `AGENTS.md` lists it. The only gap is that `README.md` does not mention it
  (line 70 lists only `k9s, kubectl`). So the kind work is documentation only.
- pi is published on npm as `@mariozechner/pi-coding-agent` and installs a `pi`
  binary. It is not in Homebrew core, so it should be installed the same way as
  Claude Code: npm global, via nvm's Node.
- The default `CLAUDE.md` template is `configs/claude/.claude/CLAUDE.sample.md`.
  `bootstrap.sh` copies it to `~/.claude/CLAUDE.md` **only if that file is missing**,
  so existing machines will not pick up the new rule automatically (see Rollout).
- The template already has a narrower rule under `# Tools Usage`:
  "For YAML file validation, always use `yq`." The new rule replaces it.
- `configs/claude/.claude/CLAUDE.md` and other runtime files under
  `configs/claude/.claude/` are untracked local state. They stay out of this change.

## Changes

### 1. kind (docs only)

- `README.md` line 70: `- k9s, kubectl (Kubernetes)` → `- k9s, kubectl, kind (Kubernetes)`.
- `README.md` "Kubernetes Tools" section: add `- **kind**: local Kubernetes clusters running in Docker (requires Docker Desktop)`.
- No `Brewfile` or `install.sh` changes.

Verify: `grep -n kind README.md Brewfile`.

### 2. pi (npm install block)

`install.sh`:

- Register the item after `claude-code`:
  ```bash
  add_item "pi"             "multi-model coding agent harness"
  ```
- Add an install block after the Claude Code block, mirroring it:
  ```bash
  # Install pi via npm
  if confirm_item "pi"; then
      export NVM_DIR="$HOME/.nvm"
      [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

      if command -v pi &> /dev/null; then
          echo "✅ pi already installed"
      elif command -v npm &> /dev/null; then
          echo "🤖 Installing pi..."
          npm install -g @mariozechner/pi-coding-agent
          echo "✅ pi installed"
      else
          echo "⚠️  Skipping pi: npm not found (install nvm first to get Node.js)"
      fi
  fi
  ```
- The `homebrew-tools` count (`22 packages via Brewfile`) is unchanged because pi is not in the Brewfile.

Docs:

- `README.md` "Other tools": add `- pi (via npm) — multi-model coding agent harness`.
- `AGENTS.md`: add "Installs pi via npm" to the install.sh list, a row in the
  "Tool Installation Details" table, and `pi` to the list of non-brew tools under
  "Script Modifications".

No pi config (`configs/pi/`) in this change. Provider keys belong in env vars or
pi's own auth, never in the repo.

Verify: `bash -n install.sh`, then `./install.sh`, answer `n` to everything except
`pi`, then run `pi --version`.

### 3. JSON/YAML rule in default CLAUDE.md

`configs/claude/.claude/CLAUDE.sample.md`, `# Tools Usage` section. Replace:

```markdown
- For YAML file validation, always use `yq`. Never write a Python script for verification.
```

with:

```markdown
- Always use `jq` to read, query, validate, or modify JSON files.
- Always use `yq` to read, query, validate, or modify YAML files.
- Never write Python scripts (including `python -c` one-liners) to process JSON or YAML.
```

The draft-PR rule stays as it is.

Docs: `README.md` "`CLAUDE.md` — global behavioral rules" list (around line 348):
add a bullet for the jq/yq rule next to the GitHub bullet.

Verify: `git diff configs/claude/.claude/CLAUDE.sample.md`.

## Rollout to existing machines

Because bootstrap does not overwrite an existing `~/.claude/CLAUDE.md`, apply the new
rule on this machine by hand (edit `~/.claude/CLAUDE.md` the same way), or delete it and re-run
`./bootstrap.sh` for the `claude` tool. This plan does not change bootstrap's copy-if-missing behaviour.

## Commits

1. `Document kind in README`
2. `Add pi coding agent to install.sh`
3. `Require jq/yq for JSON and YAML in default CLAUDE.md`

The existing uncommitted changes to `.gitconfig`, `.gitignore_global`, and `.zshrc`
are unrelated and stay out of these commits.

## Open questions

- Confirm that "pi" means `@mariozechner/pi-coding-agent` (pi-mono) and not some other tool.
- Should pi also get a stowed config (e.g. default provider/model), or install only?
