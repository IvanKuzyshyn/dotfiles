package linker

import "github.com/ivankuzyshyn/dotfiles/internal/event"

// Action re-exports event.ConflictAction for caller convenience.
type Action = event.ConflictAction

const (
	Backup    = event.ConflictBackup
	Overwrite = event.ConflictOverwrite
	Skip      = event.ConflictSkip
	Abort     = event.ConflictAbort
)

// Resolver receives a Conflict (with Target and ExistingKind) and returns
// the chosen Action. Used by Apply when handling conflicts.
type Resolver interface {
	Resolve(c Conflict) Action
}

// ResolverFunc is a function-typed adapter so callers can pass closures.
type ResolverFunc func(c Conflict) Action

func (f ResolverFunc) Resolve(c Conflict) Action { return f(c) }
