package domain

import (
	"context"
	"fmt"
	"log/slog"
)

// This file is the executable form of docs/contracts/memento-lifecycle.md.
// The transition table below and the delete rule are the single source of
// truth for which lifecycle changes are legal; the guards in the providers
// and the API handler enforce them, and every state-changing write emits a
// LogMementoStateChange line. Keep this file and the contract doc in sync.

// mementoTransitions is the legal single-step lifecycle. Same-state is always
// legal and is NOT listed here (CanTransitionMementoState handles it). Creation
// (no prior row) is unconstrained and is gated by the caller, not this table.
// archived is reserved: it appears in no entry, so nothing may enter or leave it.
var mementoTransitions = map[MementoState]map[MementoState]bool{
	MementoCandidateState: {MementoDraft: true},
	MementoDraft:          {MementoAuthored: true},
	MementoAuthored:       {MementoPublished: true},
	MementoPublished:      {MementoAuthored: true},
}

// CanTransitionMementoState reports whether moving an existing memento from one
// state to another is legal. Same-state (from == to) is always legal (no-op
// saves, re-imports at the same state). Creation is not a transition and must
// be gated by the caller (it does not call this function).
func CanTransitionMementoState(from, to MementoState) bool {
	if from == to {
		return true
	}
	return mementoTransitions[from][to]
}

// InvalidTransitionError is returned by a write path when a patch requests an
// illegal state jump on an existing memento. The API maps it to HTTP 422 with
// the issue below.
type InvalidTransitionError struct {
	From MementoState
	To   MementoState
}

func (e *InvalidTransitionError) Error() string {
	return fmt.Sprintf("illegal memento state transition %q -> %q", e.From, e.To)
}

// Issues renders the error as the API/GUI validation issue.
func (e *InvalidTransitionError) Issues() []Issue {
	return []Issue{{Field: "state", Code: CodeInvalidTransition}}
}

// CanDeleteMementoState reports whether a memento in the given state may be
// deleted. Deletion is permitted only from the non-public states
// (candidate/draft/authored); a published memento must be unpublished first,
// and archived is reserved. Deletion is a terminal action, not a transition.
func CanDeleteMementoState(state MementoState) bool {
	switch state {
	case MementoCandidateState, MementoDraft, MementoAuthored:
		return true
	default:
		return false
	}
}

// DeleteRequiresUnpublishError is returned when a delete is rejected because
// the memento is still published (or archived). The API maps it to HTTP 422.
type DeleteRequiresUnpublishError struct {
	State MementoState
}

func (e *DeleteRequiresUnpublishError) Error() string {
	return fmt.Sprintf("memento in state %q cannot be deleted; unpublish it first", e.State)
}

// Issues renders the error as the API/GUI validation issue.
func (e *DeleteRequiresUnpublishError) Issues() []Issue {
	return []Issue{{Field: "state", Code: CodeDeleteRequiresUnpublish}}
}

// Event sources for the lifecycle log's `source` field.
const (
	EventSourceUnknown  = "unknown"
	EventSourceAdminAPI = "admin-api"
	EventSourcePromote  = "promote"
	EventSourceImporter = "importer"
)

type eventSourceKey struct{}

// WithEventSource labels ctx with the origin of a state change so the lifecycle
// log can attribute it. Set once at an entry point (middleware / importer).
func WithEventSource(ctx context.Context, source string) context.Context {
	return context.WithValue(ctx, eventSourceKey{}, source)
}

// EventSource returns the label set by WithEventSource, or EventSourceUnknown.
func EventSource(ctx context.Context) string {
	if v, ok := ctx.Value(eventSourceKey{}).(string); ok && v != "" {
		return v
	}
	return EventSourceUnknown
}

// LogMementoStateChange emits the one structured debug line every state-changing
// write MUST produce (docs/contracts/memento-lifecycle.md §8). An empty `from`
// denotes creation and is logged as "(new)". A nil logger falls back to the
// default, so providers without a logger field still contribute to the stream.
func LogMementoStateChange(ctx context.Context, logger *slog.Logger, m *Memento, from, to MementoState) {
	if logger == nil {
		logger = slog.Default()
	}
	fromLabel := string(from)
	if from == "" {
		fromLabel = "(new)"
	}
	logger.InfoContext(ctx, "memento state change",
		"memento_id", m.ID,
		"journey_id", m.JourneyID,
		"from", fromLabel,
		"to", string(to),
		"revision", m.Revision,
		"source", EventSource(ctx),
	)
}
