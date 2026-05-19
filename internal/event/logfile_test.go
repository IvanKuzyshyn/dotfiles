package event_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
)

func TestLogFileSink_WritesAndKeeps(t *testing.T) {
	dir := t.TempDir()
	// Pre-create 12 old log files
	runs := filepath.Join(dir, "runs")
	if err := os.MkdirAll(runs, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		p := filepath.Join(runs, "old-"+string(rune('a'+i))+".log")
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s, err := event.NewLogFileSink(dir, 10)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.Send(event.Event{Kind: event.LogLine, Tool: "x", Step: "y", Line: "z"})

	entries, err := os.ReadDir(runs)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".log" {
			count++
		}
	}
	if count > 10 {
		t.Errorf("expected ≤10 log files after prune, got %d", count)
	}

	// Verify the new log file has the line we sent
	data, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "z") {
		t.Errorf("expected log to contain 'z', got: %s", data)
	}
}
