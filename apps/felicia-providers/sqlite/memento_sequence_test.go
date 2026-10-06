package sqlite_test

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	modsqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
)

// These tests are the provider-side regression for automatic memento
// sequence allocation (docs/contracts/automatic-sequence-allocation.md):
// new-memento creation through Repository.CreateMementoWithNextSequence
// must allocate MAX(seq)+1 within the journey atomically, keep explicit
// authored positions and tied rows untouched, and make a same-ID retry with
// matching authored values return the stored row unchanged.

const (
	mementoSeqChildEnv        = "FELICIA_MEMENTO_SEQUENCE_CHILD"
	mementoSeqTestDBEnv       = "FELICIA_MEMENTO_TEST_DB"
	mementoSeqTestJourneyEnv  = "FELICIA_MEMENTO_TEST_JOURNEY"
	mementoSeqTestIdxEnv      = "FELICIA_MEMENTO_TEST_IDX"
	mementoSeqBarrierLimit    = 30 * time.Second
	mementoSeqChildTimeoutArg = "-test.timeout=60s"
)

// seedMementoSequenceWorkspace creates a journal, a journey and two tied
// rows at seq 4 (authored explicit positions) on the given handle.
func seedMementoSequenceWorkspace(t *testing.T, repo *sqlite.Repository, slug string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	journal := &domain.Journal{ID: mustUUID(t), CreatedAt: time.Now().UTC()}
	if err := repo.CreateJournal(ctx, journal); err != nil {
		t.Fatalf("seed journal: %v", err)
	}
	journey := &domain.Journey{
		ID: mustUUID(t), JournalID: journal.ID, Slug: slug,
		Title: "Memento sequence", Place: "Kyoto",
		DateStart: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		DateEnd:   time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC),
	}
	if err := repo.UpsertJourney(ctx, journey); err != nil {
		t.Fatalf("seed journey: %v", err)
	}
	// Two explicitly authored tied rows: allocation must sit above them and
	// must never renumber them.
	for i, name := range []string{"Tie one", "Tie two"} {
		if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{
			Memento: &domain.Memento{ID: mustUUID(t), JourneyID: journey.ID, Kind: "ticket", Seq: 4, State: domain.MementoDraft, Title: name, KindData: []byte(`{}`)},
			Fields:  []string{"kind", "seq", "title", "kind_data"},
			State:   domain.MementoDraft,
		}); err != nil {
			t.Fatalf("seed tied memento %d: %v", i, err)
		}
	}
	return journey.ID
}

// sampleCreateMemento builds one full authored draft creation body. idx
// varies only the values that must differ between two real creations; the
// source identity is unique per idx so helper creations can never collide
// with each other through the (source_system, source_external_id) unique
// index.
func sampleCreateMemento(id, journeyID uuid.UUID, idx int) *domain.Memento {
	title := fmt.Sprintf("Fushimi gate %d", idx)
	amount := int64(500 + idx)
	sourceRef := fmt.Sprintf("distant-observer/%d", idx)
	orphaned := time.Date(2026, 4, 2, 12, idx, 0, 0, time.UTC)
	return &domain.Memento{
		ID: id, JourneyID: journeyID, Kind: "ticket",
		OccurredAt: time.Date(2026, 4, 2, 10, idx, 0, 0, time.UTC),
		OccurredTZ: "Asia/Tokyo",
		Geom:       orb.Point{135.77 + float64(idx)*0.01, 34.98},
		Title:      title, Place: "Kyoto",
		Vendor:         ptrString(fmt.Sprintf("Fushimi station %d", idx)),
		Essay:          ptrString(fmt.Sprintf("First stamp of day %d.", idx)),
		PriceAmount:    &amount,
		PriceCurrency:  ptrString("JPY"),
		KindData:       []byte(fmt.Sprintf(`{"entrance":"south","idx":%d}`, idx)),
		SourceIdentity: &domain.SourceIdentity{System: "sequence-test", ExternalID: fmt.Sprintf("ext-%d", idx)},
		SourceRef:      &sourceRef,
		AuthoredFields: []string{"kind", "occurred_at", "occurred_tz", "geom", "title", "place", "vendor", "essay", "price_amount", "price_currency", "kind_data"},
		OrphanedAt:     &orphaned,
		State:          domain.MementoDraft,
	}
}

func ptrString(value string) *string { return &value }

// createMementoSequences lists the persisted memento positions of a journey.
func createMementoSequences(t *testing.T, repo *sqlite.Repository, journeyID uuid.UUID) map[uuid.UUID]int {
	t.Helper()
	rows, err := repo.ListMementosByJourney(context.Background(), journeyID)
	if err != nil {
		t.Fatalf("list mementos: %v", err)
	}
	seqs := make(map[uuid.UUID]int, len(rows))
	for _, row := range rows {
		seqs[row.ID] = row.Seq
	}
	return seqs
}

func TestCreateMementoAllocatesNextSequenceAtomically(t *testing.T) {
	repo, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer repo.Close()
	journeyID := seedMementoSequenceWorkspace(t, repo, "memento-sequence")
	ctx := context.Background()

	first := sampleCreateMemento(mustUUID(t), journeyID, 1)
	created, err := repo.CreateMementoWithNextSequence(ctx, first)
	if err != nil {
		t.Fatalf("create first memento: %v", err)
	}
	if created.ID != first.ID || created.Seq != 5 || created.Revision != 1 {
		t.Fatalf("created memento = id %s seq %d revision %d, want id %s seq 5 revision 1", created.ID, created.Seq, created.Revision, first.ID)
	}
	second := sampleCreateMemento(mustUUID(t), journeyID, 2)
	created2, err := repo.CreateMementoWithNextSequence(ctx, second)
	if err != nil {
		t.Fatalf("create second memento: %v", err)
	}
	if created2.Seq != 6 || created2.Revision != 1 {
		t.Fatalf("second created seq %d revision %d, want 6 and 1", created2.Seq, created2.Revision)
	}

	// Persisted authored metadata is exactly what was supplied, field for
	// field, including the optional pointer fields and exact geometry.
	persisted, err := repo.GetMemento(ctx, first.ID)
	if err != nil {
		t.Fatalf("get created memento: %v", err)
	}
	if persisted.Kind != first.Kind || persisted.Title != first.Title || persisted.Place != first.Place ||
		persisted.OccurredAt != first.OccurredAt || persisted.OccurredTZ != first.OccurredTZ ||
		persisted.PriceAmount == nil || *persisted.PriceAmount != *first.PriceAmount ||
		persisted.PriceCurrency == nil || *persisted.PriceCurrency != *first.PriceCurrency ||
		persisted.Vendor == nil || *persisted.Vendor != *first.Vendor ||
		persisted.Essay == nil || *persisted.Essay != *first.Essay ||
		persisted.SourceIdentity == nil || *persisted.SourceIdentity != *first.SourceIdentity ||
		persisted.SourceRef == nil || *persisted.SourceRef != *first.SourceRef ||
		persisted.OrphanedAt == nil || *persisted.OrphanedAt != *first.OrphanedAt ||
		!slices.Equal(persisted.AuthoredFields, first.AuthoredFields) ||
		string(persisted.KindData) != string(first.KindData) || persisted.State != domain.MementoDraft {
		t.Fatalf("persisted authored fields drifted: %#v", persisted)
	}
	if persisted.Geom == nil {
		t.Fatalf("persisted geometry missing")
	}
	if persistedPoint, ok := persisted.Geom.(orb.Point); !ok || persistedPoint != first.Geom.(orb.Point) {
		t.Fatalf("persisted geometry = %v, want exactly %v", persisted.Geom, first.Geom)
	}
	if persisted.CreatedAt.IsZero() || persisted.UpdatedAt.IsZero() {
		t.Fatalf("creation timestamps missing: %#v", persisted)
	}
	if persisted.CreatedAt.IsZero() || persisted.UpdatedAt.IsZero() {
		t.Fatalf("creation timestamps missing: %#v", persisted)
	}

	// The tied explicit rows are untouched and the new positions are distinct.
	seqs := createMementoSequences(t, repo, journeyID)
	ties := 0
	for id, seq := range seqs {
		if id == first.ID || id == second.ID {
			continue
		}
		ties++
		if seq != 4 {
			t.Fatalf("tied row %s moved to seq %d", id, seq)
		}
	}
	if ties != 2 {
		t.Fatalf("tied rows = %d, want 2", ties)
	}
	if seqs[first.ID] != 5 || seqs[second.ID] != 6 {
		t.Fatalf("persisted positions = %d,%d, want 5,6", seqs[first.ID], seqs[second.ID])
	}

	// A retry with the same ID and matching authored values returns the
	// stored row without reallocating, updating or bumping the revision.
	retry, err := repo.CreateMementoWithNextSequence(ctx, first)
	if err != nil {
		t.Fatalf("retry create: %v", err)
	}
	if retry.ID != first.ID || retry.Seq != 5 || retry.Revision != 1 {
		t.Fatalf("retry returned id %s seq %d revision %d, want unchanged stored row", retry.ID, retry.Seq, retry.Revision)
	}
	if after := createMementoSequences(t, repo, journeyID); after[first.ID] != 5 || len(after) != 4 {
		t.Fatalf("retry mutated stored rows: %v", after)
	}

	// Same ID with a changed title is a conflict; the stored row must survive.
	changed := sampleCreateMemento(first.ID, journeyID, 1)
	changed.Title = "Renamed by a stale form"
	if _, err := repo.CreateMementoWithNextSequence(ctx, changed); !errors.Is(err, domain.ErrWriteConflict) {
		t.Fatalf("changed-title retry error = %v, want ErrWriteConflict", err)
	}
	stored, err := repo.GetMemento(ctx, first.ID)
	if err != nil || stored.Title != first.Title || stored.Revision != 1 || stored.Seq != 5 {
		t.Fatalf("conflicting retry mutated the stored row: %#v %v", stored, err)
	}

	// Same ID under a different journey is a conflict; the stored row must
	// be byte-for-byte what it was before the attempt.
	otherJourneyID := seedMementoSequenceWorkspace(t, repo, "memento-sequence-other")
	beforeConflict, err := repo.GetMemento(ctx, first.ID)
	if err != nil {
		t.Fatalf("snapshot before other-journey conflict: %v", err)
	}
	beforeOtherCount := len(createMementoSequences(t, repo, otherJourneyID))
	if _, err := repo.CreateMementoWithNextSequence(ctx, sampleCreateMemento(first.ID, otherJourneyID, 1)); !errors.Is(err, domain.ErrWriteConflict) {
		t.Fatalf("other-journey retry error = %v, want ErrWriteConflict", err)
	}
	afterConflict, err := repo.GetMemento(ctx, first.ID)
	if err != nil {
		t.Fatalf("snapshot after other-journey conflict: %v", err)
	}
	if !reflect.DeepEqual(beforeConflict, afterConflict) {
		t.Fatalf("other-journey conflict mutated the stored row:\n before: %#v\n after:  %#v", beforeConflict, afterConflict)
	}
	if afterOther := len(createMementoSequences(t, repo, otherJourneyID)); afterOther != beforeOtherCount {
		t.Fatalf("other journey rows = %d, want stable %d", afterOther, beforeOtherCount)
	}
	if original := createMementoSequences(t, repo, journeyID); len(original) != 4 {
		t.Fatalf("original journey rows = %d, want stable 4", len(original))
	}

	// An intervening authored edit makes the original create body stale.
	edited := sampleCreateMemento(second.ID, journeyID, 2)
	edited.Title = "Edited after creation"
	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{Memento: edited, Fields: []string{"title"}, ExpectedRevision: ptrInt64(1)}); err != nil {
		t.Fatalf("edit created memento: %v", err)
	}
	if _, err := repo.CreateMementoWithNextSequence(ctx, second); !errors.Is(err, domain.ErrWriteConflict) {
		t.Fatalf("stale create body error = %v, want ErrWriteConflict", err)
	}
	afterEdit, err := repo.GetMemento(ctx, second.ID)
	if err != nil || afterEdit.Title != "Edited after creation" || afterEdit.Revision != 2 {
		t.Fatalf("stale retry overwrote an edit: %#v %v", afterEdit, err)
	}

	// Explicit authored sequence changes through the old path still work,
	// including stale-revision protection.
	moved := sampleCreateMemento(second.ID, journeyID, 2)
	moved.Seq = 7
	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{Memento: moved, Fields: []string{"seq"}, ExpectedRevision: ptrInt64(2)}); err != nil {
		t.Fatalf("explicit seq update: %v", err)
	}
	if after := createMementoSequences(t, repo, journeyID); after[second.ID] != 7 {
		t.Fatalf("explicit seq update ignored: %v", after)
	}
	stale := sampleCreateMemento(second.ID, journeyID, 2)
	stale.Seq = 8
	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{Memento: stale, Fields: []string{"seq"}, ExpectedRevision: ptrInt64(2)}); !errors.Is(err, domain.ErrWriteConflict) {
		t.Fatalf("stale explicit update error = %v, want ErrWriteConflict", err)
	}

	// Invalid create inputs fail clearly.
	if _, err := repo.CreateMementoWithNextSequence(ctx, nil); err == nil {
		t.Fatalf("nil memento create succeeded")
	}
	if _, err := repo.CreateMementoWithNextSequence(ctx, sampleCreateMemento(uuid.Nil, journeyID, 3)); err == nil {
		t.Fatalf("nil-ID create succeeded")
	}
	kindless := sampleCreateMemento(mustUUID(t), journeyID, 3)
	kindless.Kind = ""
	if _, err := repo.CreateMementoWithNextSequence(ctx, kindless); err == nil {
		t.Fatalf("empty-kind create succeeded")
	}

	// The new-draft creator only ever creates drafts: a non-draft state must
	// be rejected before SQL and leave no row.
	nonDraft := sampleCreateMemento(mustUUID(t), journeyID, 3)
	nonDraft.State = domain.MementoAuthored
	if _, err := repo.CreateMementoWithNextSequence(ctx, nonDraft); err == nil {
		t.Fatalf("non-draft state create succeeded")
	}
	if _, err := repo.GetMemento(ctx, nonDraft.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("non-draft state create left a row: %v", err)
	}

	// A missing journey is a foreign-key failure, not an identity duplicate:
	// the error must propagate unchanged and never enter a successful retry.
	foreignID := mustUUID(t)
	foreign := sampleCreateMemento(foreignID, mustUUID(t), 3)
	_, foreignErr := repo.CreateMementoWithNextSequence(ctx, foreign)
	if foreignErr == nil || errors.Is(foreignErr, domain.ErrWriteConflict) {
		t.Fatalf("missing-journey create error = %v, want a propagated constraint error", foreignErr)
	}
	var sqliteErr *modsqlite.Error
	if !errors.As(foreignErr, &sqliteErr) || sqliteErr.Code() != sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY {
		t.Fatalf("missing-journey create error is not SQLITE_CONSTRAINT_FOREIGNKEY: %v", foreignErr)
	}
	if _, err := repo.GetMemento(ctx, foreignID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing-journey create left a row: %v", err)
	}

	// A canceled context must not be treated as a duplicate or a success.
	canceled := sampleCreateMemento(mustUUID(t), journeyID, 4)
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := repo.CreateMementoWithNextSequence(cancelCtx, canceled); err == nil || errors.Is(err, domain.ErrWriteConflict) {
		t.Fatalf("canceled create error = %v, want context cancellation", err)
	} else if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled create error = %v, want wrapped context.Canceled", err)
	}
	if _, err := repo.GetMemento(ctx, canceled.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("canceled create left a row: %v", err)
	}

	// After every conflicting retry above, the first creation's stored row
	// is still exactly the row that was originally persisted.
	final, err := repo.GetMemento(ctx, first.ID)
	if err != nil {
		t.Fatalf("final snapshot: %v", err)
	}
	if !reflect.DeepEqual(beforeConflict, final) {
		t.Fatalf("stored row drifted across conflict retries:\n before: %#v\n after:  %#v", beforeConflict, final)
	}
}

func ptrInt64(value int64) *int64 { return &value }

// TestMementoCreateSequenceChildProcess is the worker entry point for the
// cross-process evidence. It is skipped in ordinary runs; the parent test
// below re-executes this binary with -test.run pinned to exactly this name
// and the child environment set.
func TestMementoCreateSequenceChildProcess(t *testing.T) {
	if os.Getenv(mementoSeqChildEnv) != "1" {
		t.Skip("worker process only")
	}
	dbPath := os.Getenv(mementoSeqTestDBEnv)
	journeyID, err := uuid.Parse(os.Getenv(mementoSeqTestJourneyEnv))
	if err != nil {
		fmt.Fprintf(os.Stderr, "child parse journey: %v\n", err)
		return
	}
	idx := 0
	if _, err := fmt.Sscanf(os.Getenv(mementoSeqTestIdxEnv), "%d", &idx); err != nil {
		fmt.Fprintf(os.Stderr, "child parse idx: %v\n", err)
		return
	}
	repo, err := sqlite.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "child open sqlite: %v\n", err)
		return
	}
	defer func() { _ = repo.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), mementoSeqBarrierLimit*2)
	defer cancel()
	memento := sampleCreateMemento(mustUUID(t), journeyID, idx)

	// Announce readiness only when this process holds a live SQLite handle
	// and is about to create, then wait for the parent's GO so both writers
	// contend for the same workspace without a timing loop.
	fmt.Println("READY")
	released := make(chan struct{})
	go func() {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err == nil && strings.TrimRight(line, "\r\n") == "GO" {
			close(released)
		}
	}()
	select {
	case <-released:
	case <-ctx.Done():
		fmt.Fprintf(os.Stderr, "child barrier context: %v\n", ctx.Err())
		return
	case <-time.After(mementoSeqBarrierLimit):
		fmt.Fprintln(os.Stderr, "child barrier: release timed out")
		return
	}

	created, err := repo.CreateMementoWithNextSequence(ctx, memento)
	if err != nil {
		fmt.Fprintf(os.Stderr, "child create: %v\n", err)
		return
	}
	receipt := struct {
		ID       uuid.UUID `json:"id"`
		Seq      int       `json:"seq"`
		Revision int64     `json:"revision"`
	}{ID: created.ID, Seq: created.Seq, Revision: created.Revision}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "child encode receipt: %v\n", err)
		return
	}
	fmt.Println(string(encoded))
}

// TestMementoCreateAllocationAcrossProcesses is the contract's real-process
// evidence: two independent test executables, each with its own SQLite
// handle against the same disposable database, synchronize at a READY/GO
// barrier and must still receive distinct automatic positions above the
// tied rows, without sleeps or race-and-hope repetition.
func TestMementoCreateAllocationAcrossProcesses(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "felicia.db")

	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open seeding handle: %v", err)
	}
	journeyID := seedMementoSequenceWorkspace(t, repo, "memento-sequence")
	if err := repo.Close(); err != nil {
		t.Fatalf("close seeding handle: %v", err)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("locate test executable: %v", err)
	}

	type mementoReceipt struct {
		ID       uuid.UUID `json:"id"`
		Seq      int       `json:"seq"`
		Revision int64     `json:"revision"`
	}
	// childResult mirrors the photo harness: the single stdout owner drains
	// the pipe before Wait, publishes once through a closed done channel,
	// and stderr is read only after the join.
	type child struct {
		cmd       *exec.Cmd
		stdin     io.WriteCloser
		stderr    *bytes.Buffer
		ready     chan struct{}
		done      chan struct{}
		receipt   mementoReceipt
		receiptOK bool
		exitErr   error
	}
	children := make([]*child, 2)
	t.Cleanup(func() {
		for _, c := range children {
			if c == nil {
				continue
			}
			_ = c.stdin.Close()
			select {
			case <-c.done:
			default:
				_ = c.cmd.Process.Kill()
			}
			<-c.done
		}
	})
	for i := range children {
		cmd := exec.Command(exe, "-test.run=^TestMementoCreateSequenceChildProcess$", mementoSeqChildTimeoutArg)
		cmd.Env = append(os.Environ(),
			mementoSeqChildEnv+"=1",
			mementoSeqTestDBEnv+"="+dbPath,
			mementoSeqTestJourneyEnv+"="+journeyID.String(),
			mementoSeqTestIdxEnv+"="+fmt.Sprint(i+1),
		)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			t.Fatalf("child %d stdin: %v", i, err)
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatalf("child %d stdout: %v", i, err)
		}
		stderr := &bytes.Buffer{}
		cmd.Stderr = stderr
		if err := cmd.Start(); err != nil {
			t.Fatalf("start child %d: %v", i, err)
		}
		c := &child{cmd: cmd, stdin: stdin, stderr: stderr,
			ready: make(chan struct{}), done: make(chan struct{})}
		children[i] = c
		go func() {
			defer close(c.done)
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				if line == "READY" {
					select {
					case <-c.ready:
					default:
						close(c.ready)
					}
					continue
				}
				var receipt mementoReceipt
				if json.Unmarshal([]byte(line), &receipt) == nil && receipt.ID != uuid.Nil {
					c.receipt, c.receiptOK = receipt, true
				}
			}
			c.exitErr = cmd.Wait()
		}()
	}

	for i, c := range children {
		select {
		case <-c.ready:
		case <-c.done:
			select {
			case <-c.ready:
			default:
				t.Fatalf("child %d exited before READY: %v (stderr: %s)", i, c.exitErr, c.stderr.String())
			}
		case <-time.After(mementoSeqBarrierLimit):
			t.Fatalf("child %d did not reach the barrier in time", i)
		}
	}
	for _, c := range children {
		if _, err := io.WriteString(c.stdin, "GO\n"); err != nil {
			t.Fatalf("release child: %v", err)
		}
	}
	receipts := make([]mementoReceipt, len(children))
	for i, c := range children {
		select {
		case <-c.done:
		case <-time.After(mementoSeqBarrierLimit):
			t.Fatalf("child %d did not finish in time", i)
		}
		if c.exitErr != nil {
			t.Fatalf("child %d failed: %v (stderr: %s)", i, c.exitErr, c.stderr.String())
		}
		if !c.receiptOK {
			t.Fatalf("child %d did not report a receipt (stderr: %s)", i, c.stderr.String())
		}
		receipts[i] = c.receipt
	}

	if receipts[0].ID == receipts[1].ID {
		t.Fatalf("children returned the same identity %s", receipts[0].ID)
	}
	// Either writer may win the allocation race; both must land exactly on
	// the next two free positions above the tied rows.
	positions := map[int]bool{receipts[0].Seq: true, receipts[1].Seq: true}
	if len(positions) != 2 || !positions[5] || !positions[6] {
		t.Fatalf("persisted positions = %d,%d, want the distinct pair {5,6}", receipts[0].Seq, receipts[1].Seq)
	}
	for _, receipt := range receipts {
		if receipt.Revision != 1 {
			t.Fatalf("child creation revision = %d, want 1", receipt.Revision)
		}
	}

	// Fresh-handle verification of the persisted end state, including the
	// untouched tied rows and the retry identity of each child body.
	verify, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open verification handle: %v", err)
	}
	defer verify.Close()
	seqs := createMementoSequences(t, verify, journeyID)
	ties := 0
	for id, seq := range seqs {
		if id == receipts[0].ID || id == receipts[1].ID {
			continue
		}
		ties++
		if seq != 4 {
			t.Fatalf("tied row %s moved to seq %d", id, seq)
		}
	}
	if ties != 2 {
		t.Fatalf("tied rows = %d, want 2", ties)
	}
	for i, receipt := range receipts {
		stored, err := verify.GetMemento(context.Background(), receipt.ID)
		if err != nil {
			t.Fatalf("verify child %d row: %v", i, err)
		}
		if stored.Seq != receipt.Seq || stored.Revision != receipt.Revision || stored.State != domain.MementoDraft {
			t.Fatalf("child %d persisted row disagrees with receipt: %#v vs %+v", i, stored, receipt)
		}
		// Same-ID retry with the same authored body is still idempotent
		// after the cross-process allocation.
		body := sampleCreateMemento(receipt.ID, journeyID, i+1)
		retry, err := verify.CreateMementoWithNextSequence(context.Background(), body)
		if err != nil {
			t.Fatalf("retry child %d creation: %v", i, err)
		}
		if retry.ID != receipt.ID || retry.Seq != receipt.Seq || retry.Revision != 1 {
			t.Fatalf("retry child %d reallocated: %#v", i, retry)
		}
	}
}
