package tui

import (
	"strconv"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
)

// fakeProg records every message sent to it. It stands in for *tea.Program in
// unit tests so we don't need a real Bubble Tea event loop.
type fakeProg struct {
	mu   sync.Mutex
	msgs []tea.Msg
}

func (f *fakeProg) Send(msg tea.Msg) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.msgs = append(f.msgs, msg)
}

func (f *fakeProg) snapshot() []tea.Msg {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]tea.Msg, len(f.msgs))
	copy(out, f.msgs)
	return out
}

func newTestSink() (*Sink, *fakeProg) {
	fp := &fakeProg{}
	return &Sink{prog: fp}, fp
}

func TestSink_SendForwardsPlainEvent(t *testing.T) {
	s, fp := newTestSink()
	e := event.Event{Kind: event.ToolStarted, Tool: "git"}

	s.Send(e)

	msgs := fp.snapshot()
	if len(msgs) != 1 {
		t.Fatalf("prog received %d messages, want 1", len(msgs))
	}
	got, ok := msgs[0].(RunEventMsg)
	if !ok {
		t.Fatalf("message type = %T, want RunEventMsg", msgs[0])
	}
	if got.Event != e {
		t.Errorf("Event = %+v, want %+v", got.Event, e)
	}
}

func TestSink_SendConflictPromptStoresResolverAndForwards(t *testing.T) {
	s, fp := newTestSink()
	ch := make(chan event.ConflictAction, 1)
	e := event.Event{
		Kind: event.ConflictPrompt,
		Tool: "git",
		Conflict: &event.Conflict{
			TargetPath:   "/home/user/.gitconfig",
			ExistingKind: "file",
			Resolver:     ch,
		},
	}

	s.Send(e)

	msgs := fp.snapshot()
	if len(msgs) != 1 {
		t.Fatalf("prog received %d messages, want 1", len(msgs))
	}
	if _, ok := msgs[0].(RunEventMsg); !ok {
		t.Fatalf("message type = %T, want RunEventMsg", msgs[0])
	}

	// Resolver should now be registered under the target path.
	s.Resolve("/home/user/.gitconfig", event.ConflictBackup)
	select {
	case got := <-ch:
		if got != event.ConflictBackup {
			t.Errorf("channel got %v, want ConflictBackup", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for resolver to receive action")
	}
}

func TestSink_ResolveDeletesAfterDelivery(t *testing.T) {
	s, _ := newTestSink()
	ch := make(chan event.ConflictAction, 1)
	s.Send(event.Event{
		Kind: event.ConflictPrompt,
		Tool: "git",
		Conflict: &event.Conflict{
			TargetPath: "/x",
			Resolver:   ch,
		},
	})

	s.Resolve("/x", event.ConflictOverwrite)
	if got := <-ch; got != event.ConflictOverwrite {
		t.Fatalf("first resolve delivered %v, want ConflictOverwrite", got)
	}

	// A second call must not send another value into the channel; the entry
	// was removed via LoadAndDelete.
	s.Resolve("/x", event.ConflictAbort)
	select {
	case extra := <-ch:
		t.Fatalf("second resolve unexpectedly delivered %v", extra)
	case <-time.After(50 * time.Millisecond):
		// expected: no value, channel is empty.
	}
}

func TestSink_ResolveUnknownPathIsNoOp(t *testing.T) {
	s, _ := newTestSink()

	// Must not panic and must not block.
	done := make(chan struct{})
	go func() {
		s.Resolve("/never-registered", event.ConflictAbort)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Resolve on unknown path blocked")
	}
}

func TestSink_SendConflictPromptWithoutConflictDoesNotPanic(t *testing.T) {
	s, fp := newTestSink()
	// Defensive: Send should not panic if Conflict is nil even when Kind says
	// otherwise. We still forward the event.
	s.Send(event.Event{Kind: event.ConflictPrompt, Tool: "git"})
	if len(fp.snapshot()) != 1 {
		t.Errorf("event was not forwarded when Conflict was nil")
	}
}

// TestSink_ConcurrentSendAndResolve exercises the sync.Map under simultaneous
// access from runner-goroutine Sends and tea-goroutine Resolves. Run with
// `go test -race` to catch races.
func TestSink_ConcurrentSendAndResolve(t *testing.T) {
	s, _ := newTestSink()
	const N = 200

	var wg sync.WaitGroup
	wg.Add(2 * N)
	for i := 0; i < N; i++ {
		path := strconv.Itoa(i)
		ch := make(chan event.ConflictAction, 1)

		go func() {
			defer wg.Done()
			s.Send(event.Event{
				Kind: event.ConflictPrompt,
				Tool: "x",
				Conflict: &event.Conflict{
					TargetPath: path,
					Resolver:   ch,
				},
			})
		}()

		go func() {
			defer wg.Done()
			// Spin briefly until Send has registered the resolver. We bound
			// the wait so the test fails fast on regressions instead of
			// hanging.
			deadline := time.Now().Add(2 * time.Second)
			for {
				if _, ok := s.resolvers.Load(path); ok {
					s.Resolve(path, event.ConflictSkip)
					// Cross-talk check: the resolution must land on this
					// iteration's own channel, not some other path's.
					select {
					case got := <-ch:
						if got != event.ConflictSkip {
							t.Errorf("path %s got %v, want ConflictSkip", path, got)
						}
					case <-time.After(time.Second):
						t.Errorf("path %s: resolver channel never received", path)
					}
					return
				}
				if time.Now().After(deadline) {
					t.Errorf("resolver for %s never registered", path)
					return
				}
				time.Sleep(time.Microsecond)
			}
		}()
	}
	wg.Wait()
}
