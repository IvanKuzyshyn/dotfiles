package event

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// LogFileSink writes every event verbatim to a per-run log file under
// dir/runs/<UTC-timestamp>.log. After opening, it caps the directory at
// keepN files by deleting the oldest ones.
type LogFileSink struct {
	f    *os.File
	path string
}

// NewLogFileSink opens a new log file under dir/runs/. Caller must Close.
// keepN is the number of newest run logs to retain (older are removed).
func NewLogFileSink(dir string, keepN int) (*LogFileSink, error) {
	runs := filepath.Join(dir, "runs")
	if err := os.MkdirAll(runs, 0o755); err != nil {
		return nil, err
	}
	ts := time.Now().UTC().Format("20060102T150405Z")
	p := filepath.Join(runs, ts+".log")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	if err := pruneRuns(runs, keepN); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &LogFileSink{f: f, path: p}, nil
}

// Path returns the log file path.
func (l *LogFileSink) Path() string { return l.path }

// Close closes the log file.
func (l *LogFileSink) Close() error { return l.f.Close() }

// Send writes an event line.
func (l *LogFileSink) Send(e Event) {
	switch e.Kind {
	case LogLine:
		fmt.Fprintf(l.f, "[%s/%s] %s\n", e.Tool, e.Step, e.Line)
	case ToolStarted:
		fmt.Fprintf(l.f, "[%s] started\n", e.Tool)
	case StepStarted:
		fmt.Fprintf(l.f, "[%s/%s] step started\n", e.Tool, e.Step)
	case StepSkipped:
		fmt.Fprintf(l.f, "[%s/%s] step skipped\n", e.Tool, e.Step)
	case StepFinished:
		fmt.Fprintf(l.f, "[%s/%s] step finished\n", e.Tool, e.Step)
	case StepFailed:
		fmt.Fprintf(l.f, "[%s/%s] step failed: %v\n", e.Tool, e.Step, e.Err)
	case ToolFinished:
		fmt.Fprintf(l.f, "[%s] finished\n", e.Tool)
	case ToolFailed:
		fmt.Fprintf(l.f, "[%s] failed: %v\n", e.Tool, e.Err)
	case ToolSkipped:
		fmt.Fprintf(l.f, "[%s] skipped\n", e.Tool)
	case ConflictPrompt:
		if e.Conflict != nil {
			fmt.Fprintf(l.f, "[%s] conflict at %s (%s)\n", e.Tool, e.Conflict.TargetPath, e.Conflict.ExistingKind)
		}
	case ConflictResolved:
		fmt.Fprintf(l.f, "[%s] conflict resolved\n", e.Tool)
	}
}

func pruneRuns(dir string, keepN int) error {
	if keepN <= 0 {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	type fi struct {
		name string
		t    time.Time
	}
	var logs []fi
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".log" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		logs = append(logs, fi{e.Name(), info.ModTime()})
	}
	if len(logs) <= keepN {
		return nil
	}
	sort.Slice(logs, func(i, j int) bool { return logs[i].t.After(logs[j].t) })
	for _, l := range logs[keepN:] {
		_ = os.Remove(filepath.Join(dir, l.name))
	}
	return nil
}
