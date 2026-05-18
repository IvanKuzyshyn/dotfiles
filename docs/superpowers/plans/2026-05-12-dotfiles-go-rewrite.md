# Dotfiles Go Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the bash-based dotfiles installer (`install.sh`, `bootstrap.sh`, `lib/menu.sh`, GNU Stow) with a single Go binary `dot` that exposes a Bubble Tea TUI plus Cobra subcommands for installing tools, deploying configs, and resolving conflicts.

**Architecture:** Three layers — CLI/TUI front ends both invoke a shared `Runner` that orchestrates tools loaded from YAML manifests, executes their steps through pluggable implementations, and emits structured events to whichever sink is wired in (stream output for CLI, Bubble Tea messages for TUI). All subprocess execution and filesystem access flow through two interfaces (`Exec`, `FS`) so unit tests can substitute fakes without touching the real system. Tools run sequentially; a failed tool fails itself and its dependents but does not abort the run.

**Tech Stack:** Go 1.22+, `github.com/spf13/cobra` (CLI), `github.com/charmbracelet/bubbletea` + `lipgloss` + `bubbles` (TUI), `github.com/charmbracelet/x/exp/teatest` (TUI tests), `gopkg.in/yaml.v3` (manifests), `golangci-lint` + `gofumpt` + `goimports` (tooling), Goreleaser (release pipeline).

**Spec:** `docs/superpowers/specs/2026-05-12-dotfiles-go-rewrite-design.md`

**Branch:** Continue on `new-architecture-planning`. Commit after each task; PR when Phase 4 cutover completes.

**Progress (as of commit `240e31a`):** **Phases 1, 2, and 3 complete (Tasks 1–45).** Foundation, manifest, step (all 8 types: shell, homebrew_bootstrap, brew_package, brew_cask, brewfile, git_clone, npm_global, git_config), tool, runner, linker, CLI, and TUI layers in place. `dot list`, `dot install`, `dot update`, `dot deploy`, `dot status`, `dot version` all wired. Default `dot` invocation launches the Bubble Tea TUI (picker + runner pane + conflict modal). Linker handles symlinks with conflict resolution (`--on-conflict=backup|overwrite|skip|abort`). Per-run log files at `~/.local/state/dot/runs/`. Formatted end-of-run summary with retry hints. TUI snapshot baseline established (`make snapshot` / `make test-snapshot`). `go test ./...` green across 12 packages. **Next up: Phase 4, Task 46 (`tools/homebrew.yaml`).**

---

## Working notes

- Module path: `github.com/ivankuzyshyn/dotfiles`. All internal packages live under `internal/`.
- Every test uses fake `Exec` and fake `FS`. No test in `internal/...` calls `os/exec` directly. Real subprocesses run only under build tag `integration` (Phase 5) or in manual smoke tests at phase boundaries.
- Each task is one commit. Use `gofumpt -w . && goimports -w .` before committing. `make lint` must pass.
- The `configs/` directory layout is preserved unchanged; new manifests reference it.
- The existing `Brewfile` is reused as-is by the `brewfile` step type — packages and casks continue to live in that single file.
- The existing `.githooks/pre-commit` script stays in place; `git-hooks` manifest reconciles `core.hooksPath=.githooks` like the current `install.sh:137` does.

## File structure

```
dotfiles/
├── cmd/dot/main.go                          # Cobra entry point
├── internal/
│   ├── platform/platform.go                 # OS/arch detection
│   ├── exec/exec.go                         # Exec interface + real impl
│   ├── exec/fake.go                         # Scripted fake
│   ├── fs/fs.go                             # FS interface + real impl
│   ├── fs/fake.go                           # In-memory fake
│   ├── event/event.go                       # Event kinds + EventSink
│   ├── event/stream.go                      # StreamSink (CLI output)
│   ├── event/logfile.go                     # Per-run log writer
│   ├── manifest/schema.go                   # YAML struct definitions
│   ├── manifest/loader.go                   # Parse, validate, merge
│   ├── manifest/validate.go                 # Schema + dep cycle checks
│   ├── manifest/embed.go                    # //go:embed tools/*.yaml
│   ├── tool/tool.go                         # Tool model
│   ├── tool/registry.go                     # Embedded + overlay
│   ├── tool/select.go                       # Select, expand deps, sort
│   ├── step/step.go                         # Step interface + Env
│   ├── step/registry.go                     # type → constructor
│   ├── step/shell.go                        # Phase 1 escape-hatch step
│   ├── step/homebrew_bootstrap.go           # Phase 2
│   ├── step/brew_package.go                 # Phase 2
│   ├── step/brew_cask.go                    # Phase 2
│   ├── step/brewfile.go                     # Phase 2
│   ├── step/git_clone.go                    # Phase 2
│   ├── step/npm_global.go                   # Phase 2
│   ├── step/git_config.go                   # Phase 2
│   ├── runner/runner.go                     # Orchestrator
│   ├── runner/plan.go                       # Plan value type
│   ├── linker/linker.go                     # Walk + decide
│   ├── linker/actions.go                    # Symlink/backup/overwrite
│   ├── linker/conflict.go                   # Conflict + resolver
│   ├── cli/root.go                          # Cobra root, global flags
│   ├── cli/dotfiles_dir.go                  # --dotfiles-dir resolution
│   ├── cli/list.go
│   ├── cli/install.go
│   ├── cli/update.go
│   ├── cli/deploy.go
│   ├── cli/status.go
│   ├── cli/version.go
│   ├── cli/format.go                        # End-of-run summary
│   └── tui/
│       ├── app.go                           # Top-level tea.Model
│       ├── picker.go
│       ├── runner_pane.go
│       ├── conflict_modal.go
│       └── sink.go                          # TUISink (event → tea.Msg)
├── tools/                                   # Embedded default manifests
├── configs/                                 # UNCHANGED
├── go.mod, go.sum
├── Makefile
├── .golangci.yaml
├── .goreleaser.yaml                         # Phase 5
├── install.sh                               # Replaced in Phase 5
├── .github/workflows/
│   ├── ci.yml                               # Phase 1
│   ├── integration-macos.yml                # Phase 5
│   ├── release.yml                          # Phase 5
│   └── gitleaks.yml                         # Existing — keep
└── (DELETED in Phase 4: bootstrap.sh, lib/menu.sh, .stowrc)
```

---

## Phase 1 — Core foundation

**Working software at end of phase:** `dot list` enumerates known tools; `dot install <tool>` runs `shell`-typed steps, captures output line-by-line, continues past failures, exits with the right code. Per-run log file written. No symlinks, no TUI, no other step types.

### Task 1: Initialize Go module, Makefile, lint/format config

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `.golangci.yaml`
- Modify: `.gitignore` (add `/bin/`, `/dist/`, `*.test`, `coverage.out`)

- [x] **Step 1: Initialize Go module**
  Run: `go mod init github.com/ivankuzyshyn/dotfiles`

- [x] **Step 2: Write Makefile**
  ```make
  GO ?= go
  PKG := ./...
  BIN := bin/dot

  .PHONY: build test test-int lint fmt fmt-check tidy clean
  build:
  	$(GO) build -o $(BIN) ./cmd/dot
  test:
  	$(GO) test $(PKG)
  test-int:
  	$(GO) test -tags=integration $(PKG)
  lint:
  	golangci-lint run
  fmt:
  	gofumpt -w . && goimports -w .
  fmt-check:
  	@out=$$(gofumpt -l . && goimports -l .); \
  	if [ -n "$$out" ]; then echo "$$out"; exit 1; fi
  tidy:
  	$(GO) mod tidy
  clean:
  	rm -rf bin dist
  ```

- [x] **Step 3: Write `.golangci.yaml`**
  Enable `govet`, `staticcheck`, `errcheck`, `ineffassign`, `unused`, `revive`, `gofumpt`, `goimports`, `gocyclo` (max 15), `funlen` (max 80 lines / 60 stmts), `gosec`, `errorlint`. Disable `gci`. Set `run.timeout: 5m`.

- [x] **Step 4: Update `.gitignore`**
  Append `/bin/`, `/dist/`, `*.test`, `coverage.out`.

- [x] **Step 5: Commit**
  ```bash
  git add go.mod Makefile .golangci.yaml .gitignore
  git commit -m "Initialize Go module and tooling"
  ```

### Task 2: Add CI workflow

**Files:**
- Create: `.github/workflows/ci.yml`

- [x] **Step 1: Write workflow**
  - Trigger: `push` to `main` and pull requests.
  - Job `build`: matrix `{GOOS: [darwin, linux], GOARCH: [amd64, arm64]}` on `ubuntu-latest`; `go build ./...` (cross-compile validation).
  - Job `test`: `ubuntu-latest`; `go test ./...`; uploads coverage.
  - Job `lint`: `ubuntu-latest`; `golangci-lint-action@v6`.
  - Job `fmt`: `ubuntu-latest`; install `gofumpt` and `goimports`; run `make fmt-check`.
  - All jobs use `actions/setup-go@v5` with `go-version: '1.22'`.
  - Added `permissions: contents: read` (code-review fix).

- [x] **Step 2: Commit**
  ```bash
  git add .github/workflows/ci.yml
  git commit -m "Add CI workflow for build, test, lint, fmt-check"
  ```

### Task 3: Platform detection ✓ done in commit `5e83f8e`

**Files:**
- Create: `internal/platform/platform.go`
- Create: `internal/platform/platform_test.go`

- [x] **Step 1: Write failing test**
  ```go
  package platform_test

  import (
  	"testing"
  	"github.com/ivankuzyshyn/dotfiles/internal/platform"
  )

  func TestDetect(t *testing.T) {
  	p := platform.Detect()
  	if p.OS == "" || p.Arch == "" {
  		t.Fatalf("expected non-empty OS and Arch, got %+v", p)
  	}
  }

  func TestSupports(t *testing.T) {
  	p := platform.Platform{OS: "darwin"}
  	if !p.Supports([]string{"darwin", "linux"}) {
  		t.Errorf("darwin should be in [darwin, linux]")
  	}
  	if !p.Supports(nil) {
  		t.Errorf("empty platforms list should match any OS")
  	}
  	if p.Supports([]string{"linux"}) {
  		t.Errorf("darwin should not match [linux]")
  	}
  }
  ```

- [x] **Step 2: Run, expect FAIL (no package)**
  Run: `go test ./internal/platform/...`

- [x] **Step 3: Implement**
  ```go
  package platform

  import "runtime"

  type Platform struct {
  	OS   string
  	Arch string
  }

  func Detect() Platform { return Platform{OS: runtime.GOOS, Arch: runtime.GOARCH} }

  func (p Platform) Supports(allowed []string) bool {
  	if len(allowed) == 0 {
  		return true
  	}
  	for _, a := range allowed {
  		if a == p.OS {
  			return true
  		}
  	}
  	return false
  }
  ```

- [x] **Step 4: Run, expect PASS**
  Run: `go test ./internal/platform/...`

- [x] **Step 5: Commit**
  ```bash
  git add internal/platform/
  git commit -m "Add platform detection"
  ```

### Task 4: Exec interface and real implementation ✓ done in commits `1368d71`, `d54b92b`

**Files:**
- Create: `internal/exec/exec.go`
- Create: `internal/exec/exec_test.go`

- [ ] **Step 1: Write failing test**
  ```go
  package exec_test

  import (
  	"context"
  	"strings"
  	"testing"
  	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
  )

  type capture struct{ lines []string }
  func (c *capture) Line(s string) { c.lines = append(c.lines, s) }

  func TestRealExec_Echo(t *testing.T) {
  	c := &capture{}
  	err := xexec.Real{}.Run(context.Background(), "sh", []string{"-c", "echo hello"}, nil, c)
  	if err != nil { t.Fatal(err) }
  	if len(c.lines) == 0 || !strings.Contains(c.lines[0], "hello") {
  		t.Errorf("expected hello in output, got %v", c.lines)
  	}
  }

  func TestRealExec_NonZero(t *testing.T) {
  	err := xexec.Real{}.Run(context.Background(), "sh", []string{"-c", "exit 7"}, nil, &capture{})
  	var xe *xexec.Error
  	if err == nil { t.Fatal("expected error") }
  	if !errors.As(err, &xe) || xe.ExitCode != 7 { t.Fatalf("expected ExitCode 7, got %v", err) }
  }
  ```

- [ ] **Step 2: Run, expect FAIL**
  Run: `go test ./internal/exec/...`

- [ ] **Step 3: Implement**
  - `LineSink` interface: `Line(string)`.
  - `Exec` interface: `Run(ctx, cmd string, args, env []string, sink LineSink) error`.
  - `Error` struct: `ExitCode int`, `Stderr string`; implements `error`.
  - `Real` struct implementing `Exec`: starts `os/exec.Cmd`, pipes stdout+stderr through `bufio.Scanner`, each line → `sink.Line(line)`. On exit, if `*exec.ExitError`, wrap as `*Error{ExitCode}`. Honors `ctx` cancellation (Kill).

- [ ] **Step 4: Run, expect PASS**
  Run: `go test ./internal/exec/...`

- [ ] **Step 5: Commit**
  ```bash
  git add internal/exec/
  git commit -m "Add Exec interface and real implementation"
  ```

### Task 5: Fake Exec for tests ✓ done in commit `692a86e`

**Files:**
- Create: `internal/exec/fake.go`
- Create: `internal/exec/fake_test.go`

- [ ] **Step 1: Test the fake**
  ```go
  func TestFake_MatchByCmdAndArgs(t *testing.T) {
  	f := exec.NewFake()
  	f.Script("brew", []string{"install", "jq"}, exec.Result{Lines: []string{"==> Installing jq"}, ExitCode: 0})
  	c := &capture{}
  	if err := f.Run(context.Background(), "brew", []string{"install", "jq"}, nil, c); err != nil { t.Fatal(err) }
  	if c.lines[0] != "==> Installing jq" { t.Fatalf("got %v", c.lines) }
  }

  func TestFake_NoMatch(t *testing.T) {
  	f := exec.NewFake()
  	if err := f.Run(context.Background(), "brew", []string{"install", "jq"}, nil, &capture{}); err == nil {
  		t.Error("expected unscripted-command error")
  	}
  }
  ```

- [ ] **Step 2: Implement**
  - `Fake` stores `[]scripted{cmd, args, Result}`. `Result{Lines []string; ExitCode int; Err error}`.
  - `Script(cmd, args, result)` appends; matching is exact (cmd + args slice equal).
  - `Run` finds match, emits lines, returns nil or `*Error`. No match → return descriptive error so tests fail loudly.
  - `Calls() []Call` exposes recorded invocations for assertions.

- [ ] **Step 3: Run tests, expect PASS**
  Run: `go test ./internal/exec/...`

- [ ] **Step 4: Commit**
  ```bash
  git add internal/exec/fake.go internal/exec/fake_test.go
  git commit -m "Add scripted Exec fake for unit tests"
  ```

### Task 6: FS interface and real implementation ✓ done in commit `4fa986d`

**Files:**
- Create: `internal/fs/fs.go`
- Create: `internal/fs/fs_test.go`

- [ ] **Step 1: Write failing test**
  Tests against `t.TempDir()`: create file, `Stat`, `Symlink`, `Readlink`, `Rename`, `Remove`, `Walk` (collects paths).

- [ ] **Step 2: Run, expect FAIL**
  Run: `go test ./internal/fs/...`

- [ ] **Step 3: Implement**
  - Interface `FS` with `Stat`, `Lstat`, `Symlink`, `Readlink`, `Mkdir`, `MkdirAll`, `Rename`, `Remove`, `RemoveAll`, `Walk(root string, fn func(path string, info FileInfo, err error) error) error`.
  - `FileInfo` aliases `os.FileInfo`.
  - `Real` struct delegates to `os` and `filepath.Walk`.

- [ ] **Step 4: Run, expect PASS**
  Run: `go test ./internal/fs/...`

- [ ] **Step 5: Commit**
  ```bash
  git add internal/fs/fs.go internal/fs/fs_test.go
  git commit -m "Add FS interface and real implementation"
  ```

### Task 7: Fake FS for tests ✓ done in commit `9824e60`

**Files:**
- Create: `internal/fs/fake.go`
- Create: `internal/fs/fake_test.go`

- [ ] **Step 1: Write tests**
  Cover: `Mkdir` + `Stat`, `Symlink` + `Readlink` + `Lstat` reports symlink mode, `Rename`, `Remove`, `Walk` visits in lexical order, `Stat` on missing path returns `os.ErrNotExist`.

- [ ] **Step 2: Implement**
  In-memory tree: `nodes map[string]*node`, where `node` has `mode os.FileMode`, `target string` (for symlinks), `children []string`. Path normalization via `path.Clean`. All operations operate on absolute paths.

- [ ] **Step 3: Run tests, expect PASS**
  Run: `go test ./internal/fs/...`

- [ ] **Step 4: Commit**
  ```bash
  git add internal/fs/fake.go internal/fs/fake_test.go
  git commit -m "Add in-memory FS fake for unit tests"
  ```

### Task 8: Event types, EventSink, StreamSink, per-run log file ✓ done in commit `998576b`

**Files:**
- Create: `internal/event/event.go`
- Create: `internal/event/stream.go`
- Create: `internal/event/stream_test.go`
- Create: `internal/event/logfile.go`
- Create: `internal/event/logfile_test.go`

- [ ] **Step 1: Define event types**
  ```go
  package event

  type Level int
  const (LevelInfo Level = iota; LevelWarn; LevelError)

  type Kind int
  const (
  	ToolStarted Kind = iota
  	StepStarted
  	LogLine
  	StepSkipped
  	StepFinished
  	StepFailed
  	ToolFinished
  	ToolFailed
  	ToolSkipped
  	ConflictPrompt
  	ConflictResolved
  )

  type Event struct {
  	Kind  Kind
  	Tool  string
  	Step  string
  	Line  string
  	Level Level
  	Err   error
  	Conflict *Conflict // nil unless Kind == ConflictPrompt|Resolved
  }

  type Sink interface { Send(Event) }
  type Conflict struct {
  	TargetPath string
  	ExistingKind string // file|dir|symlink-other
  	Resolver chan<- ConflictAction
  }
  type ConflictAction int
  const (ConflictBackup ConflictAction = iota; ConflictOverwrite; ConflictSkip; ConflictAbort)
  ```

- [ ] **Step 2: Implement StreamSink**
  Writes formatted lines to an `io.Writer`: `[tool] step: line` for `LogLine`; `→ tool/step` for `StepStarted`; `✓` / `✗` / `~` for finished/failed/skipped. Color via `lipgloss` only if writer is a `*os.File` and IsTerminal. Otherwise plain.

- [ ] **Step 3: Test StreamSink**
  Capture output to `bytes.Buffer`, send a fixed event sequence, assert substring matches.

- [ ] **Step 4: Implement LogFileSink**
  Opens `~/.local/state/dot/runs/<UTC-timestamp>.log`, writes every event as a structured line, rotates by keeping the newest 10 files. `Path()` exposes the file path for end-of-run printing.

- [ ] **Step 5: Test LogFileSink**
  Use a temp dir, write events, assert file exists and contains them; create 12 fake old logs, run new run, assert only 10 newest remain.

- [ ] **Step 6: Add Tee sink**
  `Tee(sinks ...Sink) Sink` fans out events to multiple sinks. CLI uses `Tee(StreamSink, LogFileSink)`; TUI replaces StreamSink with TUISink.

- [ ] **Step 7: Run all tests, expect PASS**
  Run: `go test ./internal/event/...`

- [ ] **Step 8: Commit**
  ```bash
  git add internal/event/
  git commit -m "Add event types, StreamSink, LogFileSink, Tee"
  ```

### Task 9: Manifest schema types ✓ done in commit `fe14141`

**Files:**
- Create: `internal/manifest/schema.go`
- Create: `internal/manifest/schema_test.go`

- [ ] **Step 1: Write test**
  Parse two embedded YAML strings (one minimal, one full): assert fields populate correctly; round-trip Marshal/Unmarshal preserves data.

- [ ] **Step 2: Implement schema**
  ```go
  type Tool struct {
  	Name        string   `yaml:"name"`
  	Description string   `yaml:"description"`
  	Platforms   []string `yaml:"platforms"`
  	Tags        []string `yaml:"tags"`
  	DependsOn   []string `yaml:"depends_on"`
  	Steps       []Step   `yaml:"steps"`
  	Configs     []Config `yaml:"configs"`
  	Source      string   `yaml:"-"` // "embedded" or absolute path
  }
  type Step struct {
  	Type   string         `yaml:"type"`
  	Name   string         `yaml:"name"`
  	Fields map[string]any `yaml:",inline"`
  }
  type Config struct {
  	Source string `yaml:"source"`
  	Target string `yaml:"target"`
  }
  ```

- [ ] **Step 3: Run, expect PASS**
  Run: `go test ./internal/manifest/...`

- [ ] **Step 4: Commit**
  ```bash
  git add internal/manifest/schema.go internal/manifest/schema_test.go
  git commit -m "Add manifest schema types"
  ```

### Task 10: Manifest loader (file/dir/embedded, overlay merge, platform filter) ✓ done in commit `a6eaa96` (note: `//go:embed` rejected `..` patterns, so an extra `toolsfs.go` lives at the module root)

**Files:**
- Create: `internal/manifest/loader.go`
- Create: `internal/manifest/embed.go`
- Create: `internal/manifest/loader_test.go`
- Create: `tools/.gitkeep`

- [ ] **Step 1: Write test for LoadDir**
  Set up tempdir with two YAML files (one valid, one syntactically broken). Assert valid one loads, broken one returns parse error tagged with filename.

- [ ] **Step 2: Write test for overlay merge**
  Embedded `git.yaml` defines `description: a`. Overlay `git.yaml` in tempdir defines `description: b`. After merge, `Get("git").Description == "b"`. Source field is set to `embedded` or absolute path accordingly.

- [ ] **Step 3: Write test for platform filtering**
  Tool with `platforms: [linux]` is dropped when current platform is darwin. Tool with empty `platforms` is kept.

- [ ] **Step 4: Implement loader**
  - `LoadFile(path string) (Tool, error)`: read, yaml.Unmarshal, set Source.
  - `LoadDir(dir string) ([]Tool, error)`: glob `*.yaml`, LoadFile each, accumulate errors via `errors.Join`.
  - `LoadEmbedded() ([]Tool, error)`: uses `//go:embed all:../../tools/*.yaml` (in `embed.go`).
  - `Merge(embedded, overlay []Tool) []Tool`: overlay wins by name; preserves source field.
  - `FilterPlatform(tools []Tool, p platform.Platform) []Tool`.

- [ ] **Step 5: Run, expect PASS**
  Run: `go test ./internal/manifest/...`

- [ ] **Step 6: Commit**
  ```bash
  git add internal/manifest/loader.go internal/manifest/embed.go internal/manifest/loader_test.go tools/.gitkeep
  git commit -m "Add manifest loader with embed and overlay merge"
  ```

### Task 11: Manifest validator ✓ done in commit `276739b` (schema + dep cycle)

**Files:**
- Create: `internal/manifest/validate.go`
- Create: `internal/manifest/validate_test.go`

- [ ] **Step 1: Write tests**
  - Tool with no name → error.
  - Step with unknown `type` → error (consult `step.RegisteredTypes()` via passed-in `func() []string`).
  - `depends_on` references unknown tool → error.
  - Cycle `A→B→A` detected.
  - Multiple errors collected, not stop-on-first.

- [ ] **Step 2: Implement**
  `Validate(tools []Tool, knownTypes []string) error` returns `errors.Join(...)`. Cycle detection via DFS with white/grey/black coloring.

- [ ] **Step 3: Run, expect PASS**
  Run: `go test ./internal/manifest/...`

- [ ] **Step 4: Commit**
  ```bash
  git add internal/manifest/validate.go internal/manifest/validate_test.go
  git commit -m "Add manifest validation with dep cycle detection"
  ```

### Task 12: Step interface and Env

**Files:**
- Create: `internal/step/step.go`

- [ ] **Step 1: Define interface and Env**
  ```go
  package step

  import (
  	"context"
  	"github.com/ivankuzyshyn/dotfiles/internal/event"
  	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
  	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
  	"github.com/ivankuzyshyn/dotfiles/internal/platform"
  )

  type Env struct {
  	Exec     xexec.Exec
  	FS       xfs.FS
  	Platform platform.Platform
  	HomeDir  string
  	DotfilesDir string
  }

  type Step interface {
  	Type() string
  	Name() string
  	Check(ctx context.Context, env Env) (bool, error)
  	Run(ctx context.Context, env Env, sink event.Sink) error
  }
  ```

- [ ] **Step 2: Commit**
  ```bash
  git add internal/step/step.go
  git commit -m "Add Step interface and Env"
  ```

### Task 13: Step registry

**Files:**
- Create: `internal/step/registry.go`
- Create: `internal/step/registry_test.go`

- [ ] **Step 1: Write test**
  Register a fake constructor under `"foo"`. `Build(manifest.Step{Type:"foo", Fields: {"k":"v"}})` returns the step. Unknown type returns error. `RegisteredTypes()` includes `"foo"`.

- [ ] **Step 2: Implement**
  - `var registry = map[string]Constructor{}` package-private with `sync.RWMutex`.
  - `type Constructor func(name string, fields map[string]any) (Step, error)`.
  - `Register(typeName string, c Constructor)`.
  - `Build(s manifest.Step) (Step, error)`.
  - `RegisteredTypes() []string`.

- [ ] **Step 3: Run, expect PASS**
  Run: `go test ./internal/step/...`

- [ ] **Step 4: Commit**
  ```bash
  git add internal/step/registry.go internal/step/registry_test.go
  git commit -m "Add step constructor registry"
  ```

### Task 14: Shell step

**Files:**
- Create: `internal/step/shell.go`
- Create: `internal/step/shell_test.go`

- [ ] **Step 1: Write tests**
  Using fake Exec scripted for `sh -c "<check>"` returning various exit codes:
  - `Check` returns true when scripted exit 0; false on non-zero (no error).
  - `Run` invokes `sh -c "<install>"` and streams lines to the sink via fake's Result.
  - When `Update` field is set and `Check` reports satisfied, `Run` calls `Update` instead of `Install` (reconcile-update behavior).
  - Missing `install` field → constructor error.

- [ ] **Step 2: Implement**
  ```go
  type ShellStep struct{ name, check, install, update string }
  func newShell(name string, f map[string]any) (Step, error) { /* extract fields, validate */ }
  func (s *ShellStep) Type() string { return "shell" }
  func (s *ShellStep) Name() string { return s.name }
  func (s *ShellStep) Check(ctx context.Context, env Env) (bool, error) {
  	if s.check == "" { return false, nil } // unknown → run
  	err := env.Exec.Run(ctx, "sh", []string{"-c", s.check}, nil, nopSink{})
  	if err == nil { return true, nil }
  	var xe *xexec.Error
  	if errors.As(err, &xe) { return false, nil }
  	return false, err
  }
  func (s *ShellStep) Run(ctx context.Context, env Env, sink event.Sink) error {
  	cmd := s.install
  	if s.update != "" {
  		if ok, _ := s.Check(ctx, env); ok { cmd = s.update }
  	}
  	return env.Exec.Run(ctx, "sh", []string{"-c", cmd}, nil, lineSinkAdapter{sink, env, s.name})
  }
  func init() { Register("shell", newShell) }
  ```

- [ ] **Step 3: Run, expect PASS**
  Run: `go test ./internal/step/...`

- [ ] **Step 4: Commit**
  ```bash
  git add internal/step/shell.go internal/step/shell_test.go
  git commit -m "Add shell step type"
  ```

### Task 15: Tool model and registry

**Files:**
- Create: `internal/tool/tool.go`
- Create: `internal/tool/registry.go`
- Create: `internal/tool/registry_test.go`

- [ ] **Step 1: Write test for registry**
  Build a registry from a slice of `manifest.Tool`. Each manifest with valid steps materializes into a `Tool` with built `[]Step`. Invalid step constructor → registry returns aggregated error.

- [ ] **Step 2: Implement**
  ```go
  // tool.go
  type Tool struct {
  	Name        string
  	Description string
  	Platforms   []string
  	Tags        []string
  	DependsOn   []string
  	Steps       []step.Step
  	Configs     []manifest.Config
  	Source      string
  }
  func Build(m manifest.Tool) (*Tool, error) { /* call step.Build for each */ }

  // registry.go
  type Registry struct { byName map[string]*Tool }
  func NewRegistry(ms []manifest.Tool) (*Registry, error) { /* Build each, error-join */ }
  func (r *Registry) Get(name string) (*Tool, bool)
  func (r *Registry) All() []*Tool // returns sorted by name
  ```

- [ ] **Step 3: Run, expect PASS**
  Run: `go test ./internal/tool/...`

- [ ] **Step 4: Commit**
  ```bash
  git add internal/tool/tool.go internal/tool/registry.go internal/tool/registry_test.go
  git commit -m "Add tool model and registry"
  ```

### Task 16: Selection, dep expansion, topological sort

**Files:**
- Create: `internal/tool/select.go`
- Create: `internal/tool/select_test.go`

- [ ] **Step 1: Write tests**
  - `Select(names=[], all=true, tag="")` returns all.
  - `Select(names=[], all=false, tag="gui")` returns only tools whose Tags contain "gui".
  - `Select(names=["x"], all=false, tag="")` returns just x.
  - Unknown name → error with similar-name suggestion (use Levenshtein on package `agnivade/levenshtein` OR a hand-rolled prefix-match; keep deps minimal — use simple prefix match for v1).
  - `ExpandDeps([A], registry)` where A depends_on B returns [A, B].
  - `ExpandDeps([A], registry, noDeps=true)` returns [A] only and errors if B not already installed (deferred: Phase 2 status check).
  - `Sort([A,B,C])` where A→B and B→C returns [C, B, A] (deps first).

- [ ] **Step 2: Implement Select, ExpandDeps, Sort**
  Kahn's algorithm for sort; deterministic tie-break by name.

- [ ] **Step 3: Run, expect PASS**
  Run: `go test ./internal/tool/...`

- [ ] **Step 4: Commit**
  ```bash
  git add internal/tool/select.go internal/tool/select_test.go
  git commit -m "Add tool selection, dependency expansion, topological sort"
  ```

### Task 17: Runner core

**Files:**
- Create: `internal/runner/plan.go`
- Create: `internal/runner/runner.go`
- Create: `internal/runner/runner_test.go`

- [ ] **Step 1: Write tests with fake step impls**
  - Plan with two independent tools, both pass → both `ToolFinished`; result has 2 succeeded.
  - Plan with two tools, A fails; B independent → A `ToolFailed`, B `ToolFinished`; result has 1 succeeded, 1 failed.
  - Plan with A,B where B depends_on A; A fails → B emits `ToolSkipped` with reason; result has 1 failed, 1 skipped.
  - Step that panics → `StepFailed` with error containing "panic", run continues.
  - Cancelled context → remaining tools emit `ToolSkipped` with cancelled reason; runner returns context error.
  - Step where `Check()` returns true → `StepSkipped` event, `Run` not called.

- [ ] **Step 2: Implement**
  ```go
  type Plan struct{ Tools []*tool.Tool }
  type ToolResult struct{ Tool *tool.Tool; State State; Err error }
  type State int
  const (Succeeded State = iota; Skipped; Failed)
  type Result struct{ Tools []ToolResult }

  func Run(ctx context.Context, plan Plan, env step.Env, sink event.Sink) Result {
  	failed := map[string]struct{}{}
  	var results []ToolResult
  	for _, t := range plan.Tools {
  		select { case <-ctx.Done(): /* emit ToolSkipped, append, continue */ }
  		if depFailed(t, failed) { /* emit ToolSkipped, mark, continue */ }
  		sink.Send(event.Event{Kind: event.ToolStarted, Tool: t.Name})
  		err := runTool(ctx, t, env, sink) // panic-recovered inside
  		if err != nil { /* emit ToolFailed, append, mark, continue */ }
  		sink.Send(event.Event{Kind: event.ToolFinished, Tool: t.Name})
  	}
  	return Result{Tools: results}
  }
  ```
  `runTool` iterates steps, calls Check then Run, returns first error.

- [ ] **Step 3: Run, expect PASS**
  Run: `go test ./internal/runner/...`

- [ ] **Step 4: Commit**
  ```bash
  git add internal/runner/
  git commit -m "Add runner with fault tolerance, panic recovery, dep skip"
  ```

### Task 18: Dotfiles directory resolution

**Files:**
- Create: `internal/cli/dotfiles_dir.go`
- Create: `internal/cli/dotfiles_dir_test.go`

- [ ] **Step 1: Write tests using fake FS**
  Test the resolution chain:
  - Flag set → returned regardless of others.
  - `DOTFILES_DIR` env → returned if flag empty.
  - cwd containing `configs/` → returned.
  - `~/dotfiles` with `configs/` exists → returned.
  - None apply → returns `ErrDotfilesDirNotFound`.

- [ ] **Step 2: Implement**
  ```go
  type Resolver struct{ FS xfs.FS; Home string; Cwd string; Env func(string) string }
  func (r Resolver) Resolve(flag string) (string, error)
  ```
  Walk through the precedence chain.

- [ ] **Step 3: Run, expect PASS**
  Run: `go test ./internal/cli/...`

- [ ] **Step 4: Commit**
  ```bash
  git add internal/cli/dotfiles_dir.go internal/cli/dotfiles_dir_test.go
  git commit -m "Add dotfiles directory resolution"
  ```

### Task 19: CLI root and global flags

**Files:**
- Create: `cmd/dot/main.go`
- Create: `internal/cli/root.go`

- [ ] **Step 1: Implement Cobra root**
  ```go
  // internal/cli/root.go
  package cli

  import "github.com/spf13/cobra"

  type GlobalFlags struct {
  	NonInteractive bool
  	Verbose        bool
  	ConfigDir      string // default ~/.config/dot
  	DotfilesDir    string
  }

  func NewRoot(g *GlobalFlags) *cobra.Command {
  	root := &cobra.Command{Use: "dot", Short: "Dotfiles installer and config deployer"}
  	root.PersistentFlags().BoolVar(&g.NonInteractive, "non-interactive", false, "...")
  	root.PersistentFlags().BoolVarP(&g.Verbose, "verbose", "v", false, "...")
  	root.PersistentFlags().StringVar(&g.ConfigDir, "config-dir", defaultConfigDir(), "...")
  	root.PersistentFlags().StringVar(&g.DotfilesDir, "dotfiles-dir", "", "...")
  	return root
  }
  ```

- [ ] **Step 2: Wire main**
  ```go
  // cmd/dot/main.go
  package main

  import (
  	"os"
  	"github.com/ivankuzyshyn/dotfiles/internal/cli"
  )

  var version = "dev"

  func main() {
  	g := &cli.GlobalFlags{}
  	root := cli.NewRoot(g)
  	// subcommands attached in Task 20-22
  	if err := root.Execute(); err != nil { os.Exit(1) }
  }
  ```

- [ ] **Step 3: Build, sanity-check**
  Run: `make build && ./bin/dot --help`
  Expected: usage output with global flags listed.

- [ ] **Step 4: Commit**
  ```bash
  git add cmd/dot/main.go internal/cli/root.go
  git commit -m "Add Cobra root command and global flags"
  ```

### Task 20: `dot list` subcommand

**Files:**
- Create: `internal/cli/list.go`
- Create: `internal/cli/list_test.go`
- Modify: `cmd/dot/main.go` (attach subcommand)

- [ ] **Step 1: Write test**
  Build a registry with two tools; capture stdout; assert each appears with name, source, platforms, tags.

- [ ] **Step 2: Implement**
  - `NewListCmd(g *GlobalFlags) *cobra.Command`.
  - Loads embedded + overlay (overlay from `g.ConfigDir/tools/` and `g.DotfilesDir/tools/`).
  - Prints table (use `text/tabwriter`).

- [ ] **Step 3: Attach in main**
  ```go
  root.AddCommand(cli.NewListCmd(g))
  ```

- [ ] **Step 4: Build, smoke**
  Run: `make build && ./bin/dot list`
  Expected: empty output (no tools yet) or just the header.

- [ ] **Step 5: Commit**
  ```bash
  git add internal/cli/list.go internal/cli/list_test.go cmd/dot/main.go
  git commit -m "Add dot list command"
  ```

### Task 21: `dot install` subcommand

**Files:**
- Create: `internal/cli/install.go`
- Create: `internal/cli/install_test.go`
- Modify: `cmd/dot/main.go`

- [ ] **Step 1: Write end-to-end test**
  Use fake Exec scripted to succeed on `sh -c <whatever>`. Load a fake registry with a tool having a `shell` step. Invoke install command's `Run`, assert exit code 0 and StreamSink output contains success markers.

- [ ] **Step 2: Implement**
  - Flags: `--all`, `--tag`, `--no-deps`. Positional: tool names.
  - Loads registry (Tasks 10-11, 15), selects (Task 16), expands deps, sorts.
  - Builds `step.Env` from real Exec, real FS, `platform.Detect()`, resolved homeDir, resolved dotfilesDir.
  - Sink: `Tee(StreamSink, LogFileSink)`.
  - Runs runner. Prints end-of-run summary using a simple template (final formatter goes in Phase 2 Task 41).
  - Exit code: 0 if no failures, 1 otherwise. Pre-flight errors → exit 2.

- [ ] **Step 3: Attach in main**
  ```go
  root.AddCommand(cli.NewInstallCmd(g))
  ```

- [ ] **Step 4: Smoke**
  Run: `make build && ./bin/dot install --all`
  Expected: empty plan message + exit 0 (no tools yet).

- [ ] **Step 5: Commit**
  ```bash
  git add internal/cli/install.go internal/cli/install_test.go cmd/dot/main.go
  git commit -m "Add dot install command"
  ```

### Task 22: `dot version` subcommand

**Files:**
- Create: `internal/cli/version.go`
- Modify: `cmd/dot/main.go`

- [ ] **Step 1: Implement**
  ```go
  func NewVersionCmd(version string) *cobra.Command {
  	return &cobra.Command{Use: "version", Run: func(c *cobra.Command, _ []string) {
  		fmt.Fprintln(c.OutOrStdout(), version)
  	}}
  }
  ```
  Update Makefile `build` target to inject version: `go build -ldflags "-X main.version=$(VERSION)" ...`. `VERSION ?= dev` at top.

- [ ] **Step 2: Attach in main**
  ```go
  root.AddCommand(cli.NewVersionCmd(version))
  ```

- [ ] **Step 3: Smoke**
  Run: `make build && ./bin/dot version`
  Expected: `dev`.

- [ ] **Step 4: Commit**
  ```bash
  git add internal/cli/version.go cmd/dot/main.go Makefile
  git commit -m "Add dot version command"
  ```

### Task 23: Phase 1 smoke test with example manifest

**Files:**
- Create: `tools/example.yaml`

- [ ] **Step 1: Write a no-op manifest**
  ```yaml
  name: example
  description: Smoke-test tool, no-op
  platforms: [darwin, linux]
  steps:
    - type: shell
      name: echo
      install: 'echo "hello from dot"'
  ```

- [ ] **Step 2: Smoke checks**
  Run, in order:
  ```bash
  make build
  ./bin/dot list                       # expect 'example' listed
  ./bin/dot install example            # expect success, "hello from dot" in output
  ./bin/dot install missing-tool       # expect exit 2, unknown-tool error
  ```
  For the failure path, temporarily add a `failing` manifest with `install: 'exit 7'`:
  ```bash
  ./bin/dot install example failing    # expect example succeeds, failing fails, exit 1, summary lists failing
  rm tools/failing.yaml
  ```

- [ ] **Step 3: Commit**
  ```bash
  git add tools/example.yaml
  git commit -m "Add example tool manifest for Phase 1 smoke"
  ```

**Phase 1 complete.** Stop here for review. Run `make build test lint fmt-check` — all should pass. CI on the next push should be green.

---

## Phase 2 — Step types and Linker

**Working software at end of phase:** `dot install --all` reaches the same end state as today's `./install.sh`. `dot deploy --on-conflict=backup` produces the same symlinks as `./bootstrap.sh`. `dot status` reports per-tool state. No TUI yet.

### Task 24: `homebrew_bootstrap` step

**Files:**
- Create: `internal/step/homebrew_bootstrap.go`
- Create: `internal/step/homebrew_bootstrap_test.go`

- [ ] **Step 1: Write tests with fake Exec**
  - `Check`: scripted `command -v brew` exit 0 → satisfied true.
  - `Check`: scripted exit 1 → satisfied false.
  - `Run`: invokes curl + bash installer; lines stream to sink.

- [ ] **Step 2: Implement**
  ```go
  type HomebrewBootstrapStep struct{ name string }
  func (s *HomebrewBootstrapStep) Type() string { return "homebrew_bootstrap" }
  func (s *HomebrewBootstrapStep) Check(ctx, env) (bool, error) {
  	err := env.Exec.Run(ctx, "command", []string{"-v", "brew"}, nil, nopSink{})
  	return err == nil, nil
  }
  func (s *HomebrewBootstrapStep) Run(ctx, env, sink) error {
  	return env.Exec.Run(ctx, "sh", []string{"-c", `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`}, nil, /*adapter*/)
  }
  func init() { step.Register("homebrew_bootstrap", newHomebrewBootstrap) }
  ```

- [ ] **Step 3: Run tests, expect PASS**
  Run: `go test ./internal/step/...`

- [ ] **Step 4: Commit**
  ```bash
  git add internal/step/homebrew_bootstrap.go internal/step/homebrew_bootstrap_test.go
  git commit -m "Add homebrew_bootstrap step"
  ```

### Task 25: `brew_package` step

**Files:**
- Create: `internal/step/brew_package.go`
- Create: `internal/step/brew_package_test.go`

- [ ] **Step 1: Write tests**
  - `Check`: `brew list --formula <pkg>` exit 0 → true; exit non-zero → false.
  - `Run`: `brew install <pkg>` streams lines.
  - Constructor errors when `package` field missing.

- [ ] **Step 2: Implement**
  Mirrors homebrew_bootstrap; field `package` (string).
  Register as `brew_package`.

- [ ] **Step 3: Run tests, PASS**
- [ ] **Step 4: Commit `Add brew_package step`**

### Task 26: `brew_cask` step

**Files:**
- Create: `internal/step/brew_cask.go`
- Create: `internal/step/brew_cask_test.go`

- [ ] **Step 1: Write tests**
  Same shape as brew_package but command uses `brew install --cask <pkg>` and `brew list --cask <pkg>` for check.
- [ ] **Step 2: Implement, register as `brew_cask`**
- [ ] **Step 3: Run tests, PASS**
- [ ] **Step 4: Commit `Add brew_cask step`**

### Task 27: `brewfile` step

**Files:**
- Create: `internal/step/brewfile.go`
- Create: `internal/step/brewfile_test.go`

- [ ] **Step 1: Write tests**
  - Field `path` (relative to `Env.DotfilesDir`).
  - `Check`: `brew bundle check --file=<absPath>` exit 0 → true.
  - `Run`: `brew bundle --file=<absPath>` streams lines.
- [ ] **Step 2: Implement, register as `brewfile`**
- [ ] **Step 3: PASS**
- [ ] **Step 4: Commit `Add brewfile step`**

### Task 28: `git_clone` step

**Files:**
- Create: `internal/step/git_clone.go`
- Create: `internal/step/git_clone_test.go`

- [ ] **Step 1: Write tests**
  - Fields `url`, `dest` (dest supports `~` expansion via `Env.HomeDir`).
  - `Check`: fake FS reports dest is a directory → true.
  - `Run`: when dest exists → `git -C <dest> pull --ff-only`; when missing → `git clone <url> <dest>`. Both stream lines.
  - Missing `url` or `dest` → constructor error.
- [ ] **Step 2: Implement, register as `git_clone`**
- [ ] **Step 3: PASS**
- [ ] **Step 4: Commit `Add git_clone step`**

### Task 29: `npm_global` step

**Files:**
- Create: `internal/step/npm_global.go`
- Create: `internal/step/npm_global_test.go`

- [ ] **Step 1: Write tests**
  - Field `package`.
  - `Check`: `npm list -g --depth=0 <package>` exit 0 → true.
  - `Run`: `npm install -g <package>` streams lines.
- [ ] **Step 2: Implement, register as `npm_global`**
- [ ] **Step 3: PASS**
- [ ] **Step 4: Commit `Add npm_global step`**

### Task 30: `git_config` step

**Files:**
- Create: `internal/step/git_config.go`
- Create: `internal/step/git_config_test.go`

- [ ] **Step 1: Write tests**
  - Fields `key`, `value`, `scope` (`repo` default, also `global`).
  - `Check`: `git config --get <key>` (scoped) returns existing value; compare to `value`.
  - `Run`: `git config <scope-flag> <key> <value>`.
  - Repo scope runs `git -C <Env.DotfilesDir> config core.hooksPath .githooks` to mirror `install.sh:137`.
- [ ] **Step 2: Implement, register as `git_config`**
- [ ] **Step 3: PASS**
- [ ] **Step 4: Commit `Add git_config step`**

### Task 31: Linker walking and decision

**Files:**
- Create: `internal/linker/linker.go`
- Create: `internal/linker/linker_test.go`

- [ ] **Step 1: Write tests with fake FS**
  - Source dir `configs/git/` contains `.gitconfig`. Target `~/.gitconfig` missing → decision `Symlink`.
  - Target is already a symlink pointing at source → decision `AlreadyOk`.
  - Target is a regular file → decision `Conflict` with `ExistingKind: file`.
  - Target is a directory → `Conflict` with `ExistingKind: dir`.
  - Target is a symlink to a different path → `Conflict` with `ExistingKind: symlink-other`.
  - Decisions are batched (all conflicts for a tool collected before any apply).

- [ ] **Step 2: Implement**
  ```go
  type Decision struct{ Source, Target string; Kind DecisionKind }
  type DecisionKind int
  const (DecideSymlink DecisionKind = iota; DecideAlreadyOk; DecideConflict)

  type Plan struct{ Decisions []Decision; Conflicts []Conflict }
  func Inspect(configs []manifest.Config, env step.Env) (Plan, error)
  ```

- [ ] **Step 3: PASS**
- [ ] **Step 4: Commit `Add linker walking and decisions`**

### Task 32: Linker actions (symlink, backup, overwrite, skip)

**Files:**
- Create: `internal/linker/actions.go`
- Create: `internal/linker/conflict.go`
- Create: `internal/linker/actions_test.go`

- [ ] **Step 1: Write tests**
  - `Apply(DecisionSymlink, fs)` creates parent dirs and symlink.
  - `Apply(Backup, ...)` moves target to `~/.dotfiles_backup_<UTC-ts>/<target-with-leading-slash-stripped>`, then symlinks.
  - `Apply(Overwrite, ...)` removes target, symlinks.
  - `Apply(Skip, ...)` no-op.
  - Backup directory uses one timestamp per run (passed in).

- [ ] **Step 2: Implement**
  - `ConflictAction` constants exist in `internal/event` (Task 8). Mirror or import them here.
  - `Conflict struct{ Target string; ExistingKind string; Resolver chan ConflictAction }`.
  - `Apply(action ConflictAction, source, target, backupRoot string, fs xfs.FS) error`.

- [ ] **Step 3: PASS**
- [ ] **Step 4: Commit `Add linker actions and conflict types`**

### Task 33: Wire linker into runner

**Files:**
- Modify: `internal/runner/runner.go`
- Modify: `internal/runner/runner_test.go`

- [ ] **Step 1: Add test**
  Tool with one no-op step and one config; fake FS shows target missing → after `Run`, fake FS shows symlink at target.
  Tool with config and target conflict → if resolver picks `ConflictBackup` via test sink, backup directory and symlink both materialize.
  Tool with config and resolver picks `ConflictAbort` → tool marked `ToolFailed`.

- [ ] **Step 2: Extend runner**
  After all steps succeed for a tool and `len(tool.Configs) > 0`: call `linker.Inspect`; for each conflict emit `ConflictPrompt` event with a per-conflict resolver channel; collect actions; call `linker.Apply` for each decision. If any apply fails or any resolver returns `ConflictAbort`, the tool fails.

- [ ] **Step 3: PASS**
- [ ] **Step 4: Commit `Wire linker into runner`**

### Task 34: `--on-conflict` flag and non-interactive resolver

**Files:**
- Modify: `internal/cli/install.go`
- Create: `internal/cli/conflict_resolver.go`
- Create: `internal/cli/conflict_resolver_test.go`

- [ ] **Step 1: Write test**
  Resolver constructed with `backup` returns `ConflictBackup` for every prompt. Resolver with `abort` returns `ConflictAbort`.

- [ ] **Step 2: Implement**
  ```go
  type FlagResolver struct{ Default event.ConflictAction }
  func (r FlagResolver) Resolve(c event.Conflict) event.ConflictAction { return r.Default }
  ```
  Install command: add `--on-conflict` string flag (backup|overwrite|skip|abort); default `abort` when non-interactive. Pass resolver into sink so when `ConflictPrompt` arrives, the sink immediately writes to the resolver channel.

- [ ] **Step 3: PASS, smoke**
- [ ] **Step 4: Commit `Add --on-conflict flag and non-interactive resolver`**

### Task 35: `dot deploy` subcommand

**Files:**
- Create: `internal/cli/deploy.go`
- Modify: `cmd/dot/main.go`

- [ ] **Step 1: Implement**
  Same selection + Env setup as install, but iterates tools and calls `linker.Inspect`/`Apply` directly without running step types. `--on-conflict` flag wired identically. Useful for re-deploying configs without re-installing.

- [ ] **Step 2: Smoke**
  Run: `./bin/dot deploy --all --on-conflict=skip`
  Expected: prints decisions, no errors when no tools have configs.

- [ ] **Step 3: Commit `Add dot deploy command`**

### Task 36: `dot status` subcommand

**Files:**
- Create: `internal/cli/status.go`
- Modify: `cmd/dot/main.go`

- [ ] **Step 1: Implement**
  For each selected tool: run `Check()` on every step; classify as `installed` (all satisfied), `partial` (some satisfied, some not), `missing` (none satisfied). Print table: name, status, details (per-step satisfied/missing).

- [ ] **Step 2: Smoke**
  Run: `./bin/dot status --all`

- [ ] **Step 3: Commit `Add dot status command`**

### Task 37: `dot update` subcommand (alias)

**Files:**
- Create: `internal/cli/update.go`
- Modify: `cmd/dot/main.go`

- [ ] **Step 1: Implement**
  Alias of install: re-uses `NewInstallCmd` internals; differs only in command name and help text. Reconcile is idempotent; nothing else to special-case.

- [ ] **Step 2: Smoke**
  Run: `./bin/dot update example`

- [ ] **Step 3: Commit `Add dot update alias`**

### Task 38: End-of-run summary formatter

**Files:**
- Create: `internal/cli/format.go`
- Create: `internal/cli/format_test.go`

- [ ] **Step 1: Write tests**
  Given a `runner.Result`, formatted output contains: success/skip/fail counts, per-tool status with timing, last ~50 stderr lines on failures, retry hint (e.g. `Retry failed tools: dot install <tool>...`), and log file path.

- [ ] **Step 2: Implement**
  Pure function `Format(result runner.Result, logPath string) string`. Called from install/update commands.

- [ ] **Step 3: PASS**
- [ ] **Step 4: Commit `Add end-of-run summary formatter`**

**Phase 2 complete.** Run `make build test lint fmt-check`. Manual integration: install a small subset of the eventual tool set (e.g., add a `tools/jq.yaml` with a `brew_package` step) and run `./bin/dot install jq` to verify the brew path works end-to-end. Remove the test manifest before committing.

---

## Phase 3 — TUI

**Working software at end of phase:** running `dot` alone opens a Bubble Tea picker; selected tools run with live split-pane progress and logs; conflicts surface as a modal; `r` retries failed tools.

### Task 39: TUI app shell ✓ done in commit `f639d14`

**Files:**
- Create: `internal/tui/app.go`
- Create: `internal/tui/app_test.go`

- [x] **Step 1: Add dependencies**
  ```bash
  go get github.com/charmbracelet/bubbletea github.com/charmbracelet/lipgloss github.com/charmbracelet/bubbles
  go get github.com/charmbracelet/x/exp/teatest
  ```

- [x] **Step 2: Write top-level model**
  ```go
  type screen int
  const (screenPicker screen = iota; screenRunner; screenSummary)
  type App struct {
  	screen screen
  	picker Picker
  	runner RunnerPane
  	modal  *ConflictModal // nil when not shown
  	reg    *tool.Registry
  	env    step.Env
  	width, height int
  }
  func (a App) Init() tea.Cmd
  func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd)
  func (a App) View() string
  ```
  Routes key events to active screen; handles `KeyMsg{q/ctrl+c}` → quit; `WindowSizeMsg` → propagate to sub-models.

- [x] **Step 3: Basic teatest**
  Send `tea.WindowSizeMsg{Width:80,Height:24}` then `KeyMsg{Q}`; expect program to exit. Snapshot the initial view.

- [x] **Step 4: Commit `Add TUI app shell`**

### Task 40: Picker screen ✓ done in commit `a6857a2`

**Files:**
- Create: `internal/tui/picker.go`
- Create: `internal/tui/picker_test.go`

- [x] **Step 1: Implement**
  - Wraps Bubbles `list.Model`. Items are tools with checkbox display.
  - Keys: `space` toggle selection, `a` select all, `t` cycle tag filter, `/` filter by substring, `enter` emit `StartRunMsg{[]*tool.Tool}`.
  - Lipgloss-styled rendering: highlighted, selected, hidden by filter.

- [x] **Step 2: teatest snapshots**
  Send keys: down, space, down, space, enter. Assert emitted `StartRunMsg` lists the two selected tools.

- [x] **Step 3: Commit `Add picker screen`**

### Task 41: Runner pane (split status + logs) ✓ done in commit `63c4902`

**Files:**
- Create: `internal/tui/runner_pane.go`
- Create: `internal/tui/runner_pane_test.go`

- [x] **Step 1: Implement**
  - Left: list of tools with status icons (spinner during run, ✓/✗/~ when done).
  - Right: rolling log buffer for the currently focused tool (last ~200 lines).
  - Keys: `Tab` cycle focus, `l` open full log for finished tool (pages a `viewport`), `r` retry failed (only on screen `summary`).
  - Receives events via `RunEventMsg{event.Event}` from the TUISink.

- [x] **Step 2: teatest snapshots**
  Feed a synthetic event stream (`ToolStarted`, `LogLine`×3, `ToolFinished`); assert rendering shows expected status and last log lines.

- [x] **Step 3: Commit `Add runner pane`**

### Task 42: Conflict modal ✓ done in commit `5191a06`

**Files:**
- Create: `internal/tui/conflict_modal.go`
- Create: `internal/tui/conflict_modal_test.go`

- [x] **Step 1: Implement**
  - Triggered by `RunEventMsg{Kind: ConflictPrompt}`. Modal overlays current screen.
  - Renders: target path, existing kind, four choices (Backup / Overwrite / Skip / Abort), `Apply to remaining` toggle.
  - Selection: `↑/↓` navigate, `space` toggle apply-to-remaining, `enter` confirm — sends `ConflictResolutionMsg` containing the choice; app forwards to resolver channel via TUISink.

- [x] **Step 2: teatest**
  Send `ConflictPrompt`, send `↓ ↓ enter`; assert `ConflictResolutionMsg` payload is the third choice.

- [x] **Step 3: Commit `Add conflict modal`**

### Task 43: TUISink ✓ done in commit `c355ec2`

**Files:**
- Create: `internal/tui/sink.go`
- Create: `internal/tui/sink_test.go`

- [x] **Step 1: Implement**
  ```go
  type Sink struct{ prog *tea.Program; resolvers sync.Map /* targetPath → chan event.ConflictAction */ }
  func (s *Sink) Send(e event.Event) {
  	if e.Kind == event.ConflictPrompt {
  		s.resolvers.Store(e.Conflict.TargetPath, e.Conflict.Resolver)
  	}
  	s.prog.Send(RunEventMsg{Event: e})
  }
  func (s *Sink) Resolve(targetPath string, action event.ConflictAction) {
  	if ch, ok := s.resolvers.LoadAndDelete(targetPath); ok { ch.(chan event.ConflictAction) <- action }
  }
  ```
  App forwards user's modal choice via `s.Resolve`.

- [x] **Step 2: Test**
  Use teatest with a stubbed program; send fake event; assert the RunEventMsg lands.

- [x] **Step 3: Commit `Add TUISink event bridge`**

### Task 44: TUI default entry, runner-in-goroutine ✓ done in commit `fe360d1`

**Files:**
- Modify: `cmd/dot/main.go`
- Create: `internal/tui/launch.go`

- [x] **Step 1: Implement**
  When `os.Args` has no subcommand AND stdout is a TTY (`golang.org/x/term.IsTerminal`) AND `--non-interactive` not set: `tui.Launch(g)` instead of `root.Execute()`.
  `Launch`:
  - Build registry like CLI does.
  - Construct `tea.Program(NewApp(...))`.
  - On `StartRunMsg`: spawn goroutine running `runner.Run`; pass `TUISink`. When run completes, post a `RunCompletedMsg` to the program.
  - Run program. Exit code from `Result` like the CLI does.

- [x] **Step 2: Manual TUI smoke**
  Run: `./bin/dot`
  Expected: picker opens with the example tool. Toggle, enter, watch run.

- [x] **Step 3: Commit `Wire TUI as default entry`**

### Task 45: Phase 3 wrap-up ✓ done in commit `240e31a`

**Files:**
- Modify: `internal/tui/testdata/` (snapshots)

- [x] **Step 1: Add `make snapshot` target**
  ```make
  snapshot:
  	$(GO) test -tags=teatest -update ./internal/tui/...
  ```
- [x] **Step 2: Generate initial snapshots**
  Run: `make snapshot`
- [x] **Step 3: Commit `Add TUI snapshot baseline`**

**Phase 3 complete.** Manual TUI walkthrough with the example tool. Force a conflict scenario by hand-editing a config target; resolve via modal.

---

## Phase 4 — Migration

**Working software at end of phase:** every legacy tool exists as a manifest. New binary fully replaces bash. Migration test passes.

### Task 46: `tools/homebrew.yaml` ✓ done in commit `d274ade`

**Files:**
- Create: `tools/homebrew.yaml`

- [x] **Step 1: Write manifest**
  ```yaml
  name: homebrew
  description: Homebrew package manager + packages from Brewfile
  platforms: [darwin]
  steps:
    - type: homebrew_bootstrap
      name: bootstrap homebrew
    - type: brewfile
      name: install Brewfile packages
      path: Brewfile
  ```
- [x] **Step 2: Commit**

### Task 47: `tools/rust.yaml` ✓ done in commit `7b7b3e4`

**Files:**
- Create: `tools/rust.yaml`

- [x] **Step 1: Write manifest**
  ```yaml
  name: rust
  description: Rust toolchain via rustup
  platforms: [darwin, linux]
  steps:
    - type: shell
      name: install rustup
      check: 'command -v rustc'
      install: |
        curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
  ```
- [x] **Step 2: Commit**

### Task 48: `tools/oh-my-zsh.yaml` ✓ done in commit `9854609`

**Files:**
- Create: `tools/oh-my-zsh.yaml`

- [x] **Step 1: Write manifest**
  ```yaml
  name: oh-my-zsh
  description: Zsh framework with plugins
  platforms: [darwin, linux]
  depends_on: [homebrew]   # needs git from brew
  steps:
    - type: shell
      name: install oh-my-zsh
      check: '[ -d "$HOME/.oh-my-zsh" ]'
      install: |
        sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended
    - type: git_clone
      name: install zsh-autosuggestions
      url: https://github.com/zsh-users/zsh-autosuggestions
      dest: ~/.oh-my-zsh/custom/plugins/zsh-autosuggestions
    - type: git_clone
      name: install zsh-syntax-highlighting
      url: https://github.com/zsh-users/zsh-syntax-highlighting
      dest: ~/.oh-my-zsh/custom/plugins/zsh-syntax-highlighting
  ```
- [x] **Step 2: Commit**

### Task 49: `tools/nvm.yaml` ✓ done in commit `09a53b5`

**Files:**
- Create: `tools/nvm.yaml`

- [x] **Step 1: Write manifest**
  ```yaml
  name: nvm
  description: Node Version Manager with Node LTS
  platforms: [darwin, linux]
  steps:
    - type: shell
      name: install nvm
      check: '[ -d "$HOME/.nvm" ]'
      install: |
        curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
    - type: shell
      name: install Node LTS
      check: 'export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && command -v node'
      install: |
        export NVM_DIR="$HOME/.nvm"
        [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"
        nvm install --lts
        nvm use --lts
  ```
- [x] **Step 2: Commit**

### Task 50: `tools/claude-code.yaml` ✓ done in commit `8b28dab`

**Files:**
- Create: `tools/claude-code.yaml`

- [x] **Step 1: Write manifest**
  ```yaml
  name: claude-code
  description: AI coding assistant CLI
  platforms: [darwin, linux]
  depends_on: [nvm]
  steps:
    - type: npm_global
      name: install claude-code
      package: '@anthropic-ai/claude-code'
  ```
- [x] **Step 2: Commit**

### Task 51: `tools/git-hooks.yaml` ✓ done in commit `95a8139`

**Files:**
- Create: `tools/git-hooks.yaml`

- [x] **Step 1: Write manifest**
  ```yaml
  name: git-hooks
  description: Pre-commit gitleaks scanning for this repo
  platforms: [darwin, linux]
  depends_on: [homebrew]
  steps:
    - type: git_config
      name: enable repo hooks
      key: core.hooksPath
      value: .githooks
      scope: repo
  ```
- [x] **Step 2: Commit**

### Task 52: Config-bearing tool manifests ✓ done in commit `70655ab`

**Files:**
- Create: `tools/git.yaml`
- Create: `tools/zsh.yaml`
- Create: `tools/vim.yaml`
- Create: `tools/ghostty.yaml`
- Create: `tools/k9s.yaml`
- Create: `tools/mise.yaml`
- Create: `tools/claude.yaml` (added to cover the existing `configs/claude/` directory)

- [x] **Step 1: Write each manifest**
  Each follows the same shape. Example for `git`:
  ```yaml
  name: git
  description: Git with shared config + global gitignore
  platforms: [darwin, linux]
  depends_on: [homebrew]
  configs:
    - source: git
      target: ~
  ```
  `zsh`, `vim` similar (`target: ~`). For `ghostty`, `k9s`, `mise`, `claude` the existing layout is `configs/<tool>/.config/<tool>/...` (or `.claude/...`), so `target: ~` is correct: the linker walks each leaf under `configs/<tool>/` and creates `~/.config/<tool>/<leaf>` (or `~/.claude/<leaf>`). Note: `configs/git/.gitconfig` → symlinked to `~/.gitconfig`; the linker walks `configs/git/` and creates each leaf.
- [x] **Step 2: Commit each**
  Commits: `70655ab` (git), `92d27d0` (zsh), `082afd9` (vim), `c2adbcf` (ghostty), `8fddaed` (k9s), `c517bd7` (mise), `3beb2be` (claude).

### Task 53: Migration parity test ✓ done in commit `fb79053`

**Files:**
- Create: `internal/manifest/migration_test.go`

- [x] **Step 1: Write test**
  Pinned list of legacy tools and configs:
  ```go
  var legacyTools = []string{
  	"homebrew", "rust", "oh-my-zsh", "nvm", "claude-code", "git-hooks",
  	"git", "zsh", "vim", "ghostty", "k9s", "mise",
  }
  ```
  Test loads embedded manifests + builds registry; asserts `Get(name)` returns non-nil for every legacy tool.

- [x] **Step 2: Run, expect PASS**
  Run: `go test ./internal/manifest/...`

- [x] **Step 3: Commit `Add migration parity test`**

### Task 54: Delete bash scripts ✓ done in commit `c2785aa`

**Files:**
- Delete: `bootstrap.sh`
- Delete: `lib/menu.sh`
- Delete: `lib/` (now empty)
- Delete: `.stowrc`
- Modify: `.githooks/pre-commit` (only if it references any deleted file)

- [x] **Step 1: Verify replacement is feature-complete**
  Run smoke against the manifests:
  ```bash
  make build
  ./bin/dot status --all
  ./bin/dot install --all --on-conflict=backup
  ./bin/dot deploy --all --on-conflict=backup
  ```
- [x] **Step 2: Delete files**
  ```bash
  git rm bootstrap.sh lib/menu.sh .stowrc
  rmdir lib 2>/dev/null || true
  ```
- [x] **Step 3: Check `.githooks/pre-commit`**
  Read and confirm no references to deleted paths. If references exist, fix them.

- [x] **Step 4: Commit `Remove legacy bash scripts and .stowrc`**

### Task 55: Rewrite README ✓ done in commit `7098551`

**Files:**
- Modify: `README.md`

- [x] **Step 1: Rewrite**
  New sections:
  - Install (curl bootstrap, deferred to Phase 5 for the actual URL; placeholder note for now)
  - Usage (`dot`, `dot install`, `dot deploy`, `dot status`)
  - Adding a tool (drop YAML in `tools/` or `~/.config/dot/tools/`)
  - Configuration files (preserved `configs/` layout)
  - Tooling (Makefile targets, CI)

- [x] **Step 2: Commit `Rewrite README for dot binary`**

**Phase 4 complete.** This is a natural PR boundary. Push the branch and open a PR titled "Rewrite dotfiles installer as `dot` Go binary."

---

## Phase 5 — Distribution

**Working software at end of phase:** GitHub Releases publish cross-platform binaries; fresh-machine install is one curl line.

### Task 56: Goreleaser config ✓ done in commit `966ba64`

**Files:**
- Create: `.goreleaser.yaml`

- [x] **Step 1: Write config**
  - `builds`: single build for `./cmd/dot` with `goos: [darwin, linux]`, `goarch: [amd64, arm64]`, `ldflags: -s -w -X main.version={{.Version}}`.
  - `archives`: tar.gz, name template `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}`.
  - `checksum`: `sha256`.
  - `changelog`: group by `feat`/`fix`/`other`.

- [x] **Step 2: Dry-run locally**
  Run: `goreleaser release --snapshot --clean`
  Expected: `dist/` contains four tarballs + checksum.
  Skipped locally — installed goreleaser is v1.24 and config uses v2 schema; CI workflow validates with `version: latest`.

- [x] **Step 3: Commit `Add Goreleaser config`**

### Task 57: Release workflow ✓ done in commit `f030bfd`

**Files:**
- Create: `.github/workflows/release.yml`

- [x] **Step 1: Write workflow**
  Trigger: tag push `v*`. Steps: checkout (fetch-depth 0), setup-go 1.22, run `goreleaser/goreleaser-action@v6` with token `${{ secrets.GITHUB_TOKEN }}`.

- [x] **Step 2: Commit `Add release workflow`**

### Task 58: macOS integration workflow ✓ done in commit `f42d081`

**Files:**
- Create: `.github/workflows/integration-macos.yml`

- [x] **Step 1: Write workflow**
  Triggers: `workflow_dispatch` + nightly cron. Job on `macos-latest`: setup-go, `make test-int`. Uploads logs as artifact on failure.

- [x] **Step 2: Commit `Add macOS integration workflow`**

### Task 59: Replace `install.sh` with curl bootstrap ✓ done in commit `1252da1`

**Files:**
- Modify: `install.sh` (full rewrite)

- [x] **Step 1: Write bootstrap script**
  ```bash
  #!/usr/bin/env bash
  set -euo pipefail
  REPO="ivankuzyshyn/dotfiles"
  PREFIX="${PREFIX:-$HOME/.local/bin}"
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m)"
  case "$ARCH" in x86_64) ARCH=amd64 ;; aarch64|arm64) ARCH=arm64 ;; esac
  VER="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep -m1 '"tag_name"' | cut -d'"' -f4)"
  TARBALL="dot_${VER#v}_${OS}_${ARCH}.tar.gz"
  CHECKSUMS_URL="https://github.com/$REPO/releases/download/$VER/checksums.txt"
  TARBALL_URL="https://github.com/$REPO/releases/download/$VER/$TARBALL"
  TMP="$(mktemp -d)"
  trap 'rm -rf "$TMP"' EXIT
  curl -fsSL "$TARBALL_URL" -o "$TMP/$TARBALL"
  curl -fsSL "$CHECKSUMS_URL" -o "$TMP/checksums.txt"
  (cd "$TMP" && grep " $TARBALL\$" checksums.txt | shasum -a 256 -c -)
  tar -xzf "$TMP/$TARBALL" -C "$TMP"
  mkdir -p "$PREFIX"
  install -m 0755 "$TMP/dot" "$PREFIX/dot"
  echo "Installed dot to $PREFIX/dot"
  case ":$PATH:" in *":$PREFIX:"*) ;; *) echo "Add to PATH: export PATH=\"$PREFIX:\$PATH\"" ;; esac
  ```

- [x] **Step 2: Update README install section**
  ```
  curl -fsSL https://github.com/ivankuzyshyn/dotfiles/releases/latest/download/install.sh | sh
  ```
  Make sure Goreleaser publishes `install.sh` as a release asset (`extra_files` in `.goreleaser.yaml`); update goreleaser config accordingly.

- [ ] **Step 3: Tag a pre-release** *(manual user action — out of scope for this commit)*
  Manually: `git tag v0.0.1-rc1 && git push origin v0.0.1-rc1`. Wait for release workflow. On a fresh user account or VM, run the curl line and verify `dot version` prints `0.0.1-rc1`.

- [x] **Step 4: Commit `Replace install.sh with curl bootstrap`**

**Phase 5 complete.** Ship.

---

## Self-review notes

**Spec coverage:** every requirement in `docs/superpowers/specs/2026-05-12-dotfiles-go-rewrite-design.md` maps to a task above. Notable mappings:
- "Self-contained binaries for macOS and Linux" → Phase 5 (Tasks 56-59).
- "Easy to add new tool" → Tasks 9, 10, 16 (manifest loader + overlay).
- "Install everything from scratch / specific tool" → Task 21 (CLI install command).
- "Fault tolerance" → Task 17 (runner with dep-skip + panic recovery).
- "Tools with config files + conflict resolution" → Tasks 31-34.
- "Great UI/UX" → Phase 3 (TUI).
- "Easy unit and integration tests" → Tasks 4-7 (Exec/FS seams).
- "Good local + CI setup" → Tasks 1-2.
- "Per-run log file" → Task 8 (LogFileSink).
- "Update semantics" → Task 37 (alias of install).
- "Status command (Check-only)" → Task 36.
- "`--no-deps` flag" → Task 16.
- "Embedded + overlay manifests" → Tasks 10, 20.

**Type consistency:** `Step.Check(ctx, env) (bool, error)` and `Step.Run(ctx, env, sink) error` signatures consistent across Tasks 12, 14, 24-30. `event.Event`, `event.Sink`, `event.ConflictAction` defined once in Task 8 and reused in Tasks 32-34, 42-43. `step.Env` defined in Task 12 and consumed unchanged thereafter.

**Placeholder scan:** no "TBD", no "implement later". One soft area was the rust manifest (Task 47) and nvm manifest (Task 49) — both now show the exact flags they need (preserving `install.sh:55` and `install.sh:97` respectively).

---

## Execution

Plan complete and saved to `docs/superpowers/plans/2026-05-12-dotfiles-go-rewrite.md`. Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration. Best for a multi-day rewrite of this size.

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints. Risky for a plan this large — context will get heavy by Phase 3.

Recommend **Subagent-Driven** with a checkpoint at each Phase boundary.
