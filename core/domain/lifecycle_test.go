package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/google/uuid"
)

func TestCanTransitionMementoState(t *testing.T) {
	states := []MementoState{
		MementoCandidateState, MementoDraft, MementoAuthored, MementoPublished, MementoArchived,
	}
	legal := map[MementoState]MementoState{
		MementoCandidateState: MementoDraft,
		MementoDraft:          MementoAuthored,
		MementoAuthored:       MementoPublished,
		MementoPublished:      MementoAuthored,
	}

	for _, from := range states {
		for _, to := range states {
			want := from == to || legal[from] == to
			if got := CanTransitionMementoState(from, to); got != want {
				t.Errorf("CanTransitionMementoState(%q, %q) = %v, want %v", from, to, got, want)
			}
		}
	}

	// archived is reserved: nothing enters or leaves it (same-state aside).
	for _, s := range states {
		if s == MementoArchived {
			continue
		}
		if CanTransitionMementoState(s, MementoArchived) {
			t.Errorf("transition into archived from %q must be illegal", s)
		}
		if CanTransitionMementoState(MementoArchived, s) {
			t.Errorf("transition out of archived to %q must be illegal", s)
		}
	}
}

func TestCanDeleteMementoState(t *testing.T) {
	cases := map[MementoState]bool{
		MementoCandidateState: true,
		MementoDraft:          true,
		MementoAuthored:       true,
		MementoPublished:      false,
		MementoArchived:       false,
	}
	for state, want := range cases {
		if got := CanDeleteMementoState(state); got != want {
			t.Errorf("CanDeleteMementoState(%q) = %v, want %v", state, got, want)
		}
	}
}

func TestInvalidTransitionErrorIssues(t *testing.T) {
	err := &InvalidTransitionError{From: MementoPublished, To: MementoDraft}
	issues := err.Issues()
	if len(issues) != 1 || issues[0].Field != "state" || issues[0].Code != CodeInvalidTransition {
		t.Fatalf("unexpected issues: %#v", issues)
	}
}

func TestDeleteRequiresUnpublishErrorIssues(t *testing.T) {
	err := &DeleteRequiresUnpublishError{State: MementoPublished}
	issues := err.Issues()
	if len(issues) != 1 || issues[0].Field != "state" || issues[0].Code != CodeDeleteRequiresUnpublish {
		t.Fatalf("unexpected issues: %#v", issues)
	}
}

func TestEventSource(t *testing.T) {
	if got := EventSource(context.Background()); got != EventSourceUnknown {
		t.Errorf("default EventSource = %q, want %q", got, EventSourceUnknown)
	}
	ctx := WithEventSource(context.Background(), EventSourceImporter)
	if got := EventSource(ctx); got != EventSourceImporter {
		t.Errorf("EventSource = %q, want %q", got, EventSourceImporter)
	}
}

func TestLogMementoStateChange(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	m := &Memento{ID: uuid.New(), JourneyID: uuid.New(), Revision: 3}
	ctx := WithEventSource(context.Background(), EventSourceAdminAPI)

	LogMementoStateChange(ctx, logger, m, MementoAuthored, MementoPublished)

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log line is not JSON: %v (%s)", err, buf.String())
	}
	for _, key := range []string{"memento_id", "journey_id", "from", "to", "revision", "source"} {
		if _, ok := line[key]; !ok {
			t.Errorf("log line missing key %q: %v", key, line)
		}
	}
	if line["from"] != "authored" || line["to"] != "published" || line["source"] != EventSourceAdminAPI {
		t.Errorf("unexpected log fields: %v", line)
	}

	// Creation logs from as "(new)".
	buf.Reset()
	LogMementoStateChange(context.Background(), logger, m, "", MementoDraft)
	_ = json.Unmarshal(buf.Bytes(), &line)
	if line["from"] != "(new)" {
		t.Errorf("creation from = %v, want (new)", line["from"])
	}
}
