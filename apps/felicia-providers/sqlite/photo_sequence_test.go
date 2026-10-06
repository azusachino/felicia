package sqlite_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-core/ports"
	"github.com/azusachino/felicia/apps/felicia-providers/local"
	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
	"github.com/azusachino/felicia/apps/felicia-runtime/photos"
)

// These tests are the deterministic regression for the automatic photo
// sequence contract: distinct uploads to the same workspace must each receive
// a distinct automatic position above the existing maximum, whether the
// writers share one process (two SQLite handles) or are separate OS
// processes. The barrier pins every writer between storing its original bytes
// and creating its row, so a non-atomic allocation would deterministically
// duplicate a position.

const (
	photoSeqChildEnv      = "FELICIA_PHOTO_SEQUENCE_CHILD"
	photoSeqTestDBEnv     = "FELICIA_TEST_DB"
	photoSeqTestMediaEnv  = "FELICIA_TEST_MEDIA"
	photoSeqTestMementoID = "FELICIA_TEST_MEMENTO"
	photoSeqTestShade     = "FELICIA_TEST_SHADE"
	photoSeqBarrierLimit  = 30 * time.Second
)

// pngHelper is the minimal testing surface the PNG helper needs.
type pngHelper interface {
	Helper()
	Fatalf(format string, args ...any)
}

// barrierBlob wraps the real local blob store and pins the caller after Put
// until the test releases it. Timeouts guarantee a missing release returns an
// error instead of deadlocking, and callers join via the test cleanup.
type barrierBlob struct {
	inner   ports.BlobStore
	ready   chan<- struct{}
	release <-chan struct{}
}

func (b barrierBlob) Put(ctx context.Context, key string, data []byte) error {
	if err := b.inner.Put(ctx, key, data); err != nil {
		return err
	}
	select {
	case b.ready <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(photoSeqBarrierLimit):
		return errors.New("photo sequence barrier: announcing ready timed out")
	}
	select {
	case <-b.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(photoSeqBarrierLimit):
		return errors.New("photo sequence barrier: release timed out")
	}
}

func (b barrierBlob) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	return b.inner.Open(ctx, key)
}

func (b barrierBlob) Delete(ctx context.Context, key string) error {
	return b.inner.Delete(ctx, key)
}

func sequencePNG(t pngHelper, shade byte) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetGray(x, y, color.Gray{Y: shade})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

type photoReceipt struct {
	ID        uuid.UUID `json:"id"`
	Seq       int       `json:"seq"`
	ObjectKey string    `json:"object_key"`
}

// seedPhotoSequenceWorkspace creates journal, journey, memento, the existing
// seq-4 photo row and its original bytes on the given handle.
func seedPhotoSequenceWorkspace(t *testing.T, repo *sqlite.Repository, blobRoot string, mementoID uuid.UUID) (*domain.MementoPhoto, []byte) {
	t.Helper()
	ctx := context.Background()
	journal := &domain.Journal{ID: mustUUID(t), CreatedAt: time.Now().UTC()}
	if err := repo.CreateJournal(ctx, journal); err != nil {
		t.Fatalf("seed journal: %v", err)
	}
	journey := &domain.Journey{
		ID: mustUUID(t), JournalID: journal.ID, Slug: "photo-sequence",
		Title: "Photo sequence", Place: "Tokyo",
		DateStart: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		DateEnd:   time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC),
	}
	if err := repo.UpsertJourney(ctx, journey); err != nil {
		t.Fatalf("seed journey: %v", err)
	}
	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{
		Memento: &domain.Memento{ID: mementoID, JourneyID: journey.ID, Kind: "live", Seq: 1, State: domain.MementoDraft, KindData: []byte(`{}`)},
		Fields:  []string{"kind", "seq", "kind_data"},
		State:   domain.MementoDraft,
	}); err != nil {
		t.Fatalf("seed memento: %v", err)
	}
	existing := &domain.MementoPhoto{ID: mustUUID(t), MementoID: mementoID,
		ObjectKey: "existing/original.png", ContentHash: "sha256:existing-original", Seq: 4, CreatedAt: time.Now().UTC()}
	if err := repo.UpsertPhoto(ctx, existing); err != nil {
		t.Fatalf("seed existing photo: %v", err)
	}
	existingBytes := []byte("existing original bytes")
	if err := local.NewFileBlobStore(blobRoot).Put(ctx, existing.ObjectKey, existingBytes); err != nil {
		t.Fatalf("seed existing original: %v", err)
	}
	return existing, existingBytes
}

// verifyPhotoSequence checks the persisted end state through a fresh handle:
// one row per upload plus the existing row, distinct new positions above the
// existing maximum, byte-identical originals, and the pre-existing row and
// bytes untouched.
func verifyPhotoSequence(t *testing.T, dbPath, blobRoot string, mementoID uuid.UUID, receipts []photoReceipt, payloads [][]byte, existing *domain.MementoPhoto, existingBytes []byte) {
	t.Helper()
	ctx := context.Background()
	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open verification handle: %v", err)
	}
	defer func() { _ = repo.Close() }()
	blobs := local.NewFileBlobStore(blobRoot)

	rows, err := repo.ListPhotosByMemento(ctx, mementoID)
	if err != nil {
		t.Fatalf("list persisted photos: %v", err)
	}
	if len(rows) != 1+len(receipts) {
		t.Fatalf("persisted photo rows = %d, want %d", len(rows), 1+len(receipts))
	}
	seenSeq := map[int]bool{existing.Seq: true}
	for _, receipt := range receipts {
		if seenSeq[receipt.Seq] {
			t.Errorf("duplicate persisted sequence %d", receipt.Seq)
		}
		seenSeq[receipt.Seq] = true
		if receipt.Seq <= existing.Seq {
			t.Errorf("position %d is not above the existing maximum %d", receipt.Seq, existing.Seq)
		}
		row, err := repo.GetPhoto(ctx, receipt.ID)
		if err != nil {
			t.Fatalf("get persisted photo %s: %v", receipt.ID, err)
		}
		if row.MementoID != mementoID || row.Seq != receipt.Seq || row.ObjectKey != receipt.ObjectKey {
			t.Errorf("persisted row %+v disagrees with receipt %+v", row, receipt)
		}
	}
	for seq := existing.Seq + 1; seq <= existing.Seq+len(receipts); seq++ {
		if !seenSeq[seq] {
			t.Errorf("missing next automatic position %d", seq)
		}
	}
	for i, payload := range payloads {
		reader, err := blobs.Open(ctx, receipts[i].ObjectKey)
		if err != nil {
			t.Fatalf("open original %d: %v", i, err)
		}
		stored, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatalf("read original %d: %v", i, err)
		}
		if !bytes.Equal(stored, payload) {
			t.Errorf("upload %d original bytes changed", i)
		}
	}
	row, err := repo.GetPhoto(ctx, existing.ID)
	if err != nil {
		t.Fatalf("reload existing photo: %v", err)
	}
	if row.Seq != existing.Seq || row.ContentHash != existing.ContentHash || row.ObjectKey != existing.ObjectKey {
		t.Errorf("existing photo row changed: %+v", row)
	}
	reader, err := blobs.Open(ctx, existing.ObjectKey)
	if err != nil {
		t.Fatalf("open existing original: %v", err)
	}
	stored, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatalf("read existing original: %v", err)
	}
	if !bytes.Equal(stored, existingBytes) {
		t.Error("existing original bytes changed")
	}
}

// TestConcurrentPhotoUploadAllocatesDistinctPositions proves the allocation
// across two independently opened SQLite handles in this process: both
// writers finish storing their originals before either row is created, and
// the provider must still assign distinct positions above the existing
// maximum.
func TestConcurrentPhotoUploadAllocatesDistinctPositions(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "felicia.db")
	blobRoot := filepath.Join(dir, "media")

	repoA, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open first sqlite handle: %v", err)
	}
	defer func() { _ = repoA.Close() }()
	mementoID := mustUUID(t)
	existing, existingBytes := seedPhotoSequenceWorkspace(t, repoA, blobRoot, mementoID)

	repoB, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open second sqlite handle: %v", err)
	}
	defer func() { _ = repoB.Close() }()

	localStore := local.NewFileBlobStore(blobRoot)
	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseAll := func() { releaseOnce.Do(func() { close(release) }) }
	blobs := [2]barrierBlob{{inner: localStore, ready: ready, release: release}, {inner: localStore, ready: ready, release: release}}
	uploads := [2]*photos.Service{photos.New(repoA, blobs[0]), photos.New(repoB, blobs[1])}
	payloads := [2][]byte{sequencePNG(t, 0x3C), sequencePNG(t, 0xC3)}

	type outcome struct {
		receipt photoReceipt
		err     error
	}
	results := [2]outcome{}
	var workers sync.WaitGroup
	workers.Add(2)
	for i := range uploads {
		go func(i int) {
			defer workers.Done()
			photo, err := uploads[i].Upload(context.Background(), mementoID, payloads[i])
			if err == nil {
				results[i].receipt = photoReceipt{ID: photo.ID, Seq: photo.Seq, ObjectKey: photo.ObjectKey}
			}
			results[i].err = err
		}(i)
	}
	// Release and join the writers before the repository handles close:
	// deferred functions run LIFO, so this defer — registered after the repo
	// close defers — runs first and an early failure cannot close a repo under
	// an active upload or leak a blocked goroutine into TempDir cleanup.
	defer func() {
		releaseAll()
		workers.Wait()
	}()
	for range 2 {
		select {
		case <-ready:
		case <-time.After(photoSeqBarrierLimit):
			t.Fatal("barrier: uploads did not reach original storage in time")
		}
	}
	releaseAll()
	workers.Wait()

	for i, result := range results {
		if result.err != nil {
			t.Fatalf("upload %d failed: %v", i, result.err)
		}
		if result.receipt.ID == uuid.Nil {
			t.Fatalf("upload %d returned no identity", i)
		}
	}
	if results[0].receipt.ID == results[1].receipt.ID {
		t.Fatalf("uploads returned the same identity %s", results[0].receipt.ID)
	}
	receipts := []photoReceipt{results[0].receipt, results[1].receipt}
	verifyPhotoSequence(t, dbPath, blobRoot, mementoID, receipts, payloads[:], existing, existingBytes)
}

// runPhotoSequenceChild is the entry point for one writer OS process. It
// stores its distinct original through the real local blob store, prints
// READY, waits for the parent's GO line on stdin, then completes the upload
// and prints a JSON receipt on stdout.
func runPhotoSequenceChild() int {
	ctx, cancel := context.WithTimeout(context.Background(), photoSeqBarrierLimit*2)
	defer cancel()
	repo, err := sqlite.Open(os.Getenv(photoSeqTestDBEnv))
	if err != nil {
		fmt.Fprintf(os.Stderr, "child open sqlite: %v\n", err)
		return 1
	}
	defer func() { _ = repo.Close() }()
	mementoID, err := uuid.Parse(os.Getenv(photoSeqTestMementoID))
	if err != nil {
		fmt.Fprintf(os.Stderr, "child parse memento: %v\n", err)
		return 1
	}
	shade := byte(0)
	if _, err := fmt.Sscanf(os.Getenv(photoSeqTestShade), "%02x", &shade); err != nil {
		fmt.Fprintf(os.Stderr, "child parse shade: %v\n", err)
		return 1
	}
	barrier := childBarrier{inner: local.NewFileBlobStore(os.Getenv(photoSeqTestMediaEnv))}
	photo, err := photos.New(repo, barrier).Upload(ctx, mementoID, sequencePNG(childTesting{}, shade))
	if err != nil {
		fmt.Fprintf(os.Stderr, "child upload: %v\n", err)
		return 1
	}
	receipt := photoReceipt{ID: photo.ID, Seq: photo.Seq, ObjectKey: photo.ObjectKey}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "child encode receipt: %v\n", err)
		return 1
	}
	fmt.Println(string(encoded))
	return 0
}

// childBarrier is the worker-process half of the barrier: it announces READY
// on stdout after the real Put and blocks until the parent writes GO.
type childBarrier struct{ inner ports.BlobStore }

func (b childBarrier) Put(ctx context.Context, key string, data []byte) error {
	if err := b.inner.Put(ctx, key, data); err != nil {
		return err
	}
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
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(photoSeqBarrierLimit):
		return errors.New("photo sequence child barrier: release timed out")
	}
}

func (b childBarrier) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	return b.inner.Open(ctx, key)
}

func (b childBarrier) Delete(ctx context.Context, key string) error {
	return b.inner.Delete(ctx, key)
}

// childTesting adapts the worker (outside any testing.T) to the PNG helper.
type childTesting struct{}

func (childTesting) Helper()                           {}
func (childTesting) Fatalf(format string, args ...any) { panic(fmt.Sprintf(format, args...)) }

func TestMain(m *testing.M) {
	if os.Getenv(photoSeqChildEnv) == "1" {
		os.Exit(runPhotoSequenceChild())
	}
	os.Exit(m.Run())
}

// TestPhotoSequenceAllocationAcrossProcesses is the cross-process evidence
// the contract requires: two real child OS processes, each running this
// race-instrumented test binary in worker mode against the same disposable
// workspace, synchronize only after their original bytes are stored and must
// still receive distinct automatic positions.
func TestPhotoSequenceAllocationAcrossProcesses(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "felicia.db")
	blobRoot := filepath.Join(dir, "media")

	repoA, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open seeding handle: %v", err)
	}
	mementoID := mustUUID(t)
	existing, existingBytes := seedPhotoSequenceWorkspace(t, repoA, blobRoot, mementoID)
	if err := repoA.Close(); err != nil {
		t.Fatalf("close seeding handle: %v", err)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("locate test executable: %v", err)
	}
	shades := []byte{0x3C, 0xC3}
	// childResult publishes each child's outcome exactly once: the owner
	// goroutine drains stdout fully before calling Wait (StdoutPipe must not
	// be closed concurrently with a live scanner), then stores the receipt and
	// exit error and closes done. done is closed only once and is safe to
	// receive from repeatedly, so cleanup and the parent never consume the
	// exit result twice. stderr is read only after done, so the buffer is
	// final when it is reported.
	type child struct {
		cmd       *exec.Cmd
		stdin     io.WriteCloser
		stderr    *bytes.Buffer
		ready     chan struct{} // closed once when READY is seen on stdout
		done      chan struct{} // closed once after stdout drained and Wait returned
		receipt   photoReceipt
		receiptOK bool
		exitErr   error
	}
	children := make([]*child, len(shades))
	t.Cleanup(func() {
		// Close stdin, kill only the exact owned children, and join every
		// owner goroutine on every exit path, before TempDir removal: an early
		// failure must not leave live writers against a deleted workspace.
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
	for i, shade := range shades {
		cmd := exec.Command(exe)
		cmd.Env = append(os.Environ(),
			photoSeqChildEnv+"=1",
			photoSeqTestDBEnv+"="+dbPath,
			photoSeqTestMediaEnv+"="+blobRoot,
			photoSeqTestMementoID+"="+mementoID.String(),
			photoSeqTestShade+"="+fmt.Sprintf("%02x", shade),
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
				var receipt photoReceipt
				if json.Unmarshal([]byte(line), &receipt) == nil && receipt.ID != uuid.Nil {
					c.receipt, c.receiptOK = receipt, true
				}
			}
			c.exitErr = cmd.Wait()
		}()
	}

	// Wait for READY from every child: both originals are stored before any
	// row creation runs, so a non-atomic allocation would deterministically
	// duplicate the next position.
	for i, c := range children {
		select {
		case <-c.ready:
		case <-c.done:
			select {
			case <-c.ready:
			default:
				t.Fatalf("child %d exited before READY: %v (stderr: %s)", i, c.exitErr, c.stderr.String())
			}
		case <-time.After(photoSeqBarrierLimit):
			t.Fatalf("child %d did not reach original storage in time", i)
		}
	}
	for i, c := range children {
		if _, err := io.WriteString(c.stdin, "GO\n"); err != nil {
			t.Fatalf("release child %d: %v", i, err)
		}
		_ = c.stdin.Close()
	}

	receipts := make([]photoReceipt, len(children))
	for i, c := range children {
		select {
		case <-c.done:
			if !c.receiptOK {
				t.Fatalf("child %d exited before receipt: %v (stderr: %s)", i, c.exitErr, c.stderr.String())
			}
		case <-time.After(photoSeqBarrierLimit):
			t.Fatalf("child %d did not report a receipt in time", i)
		}
		// In the done case the child is already joined (done implies Wait has
		// returned); in the timeout case t.Fatalf defers to cleanup, which
		// closes stdin, kills and joins every owned child.
		if c.exitErr != nil {
			t.Fatalf("child %d exit: %v (stderr: %s)", i, c.exitErr, c.stderr.String())
		}
		receipts[i] = c.receipt
	}
	if receipts[0].ID == receipts[1].ID {
		t.Fatalf("children returned the same identity %s", receipts[0].ID)
	}
	if receipts[0].Seq == receipts[1].Seq {
		t.Errorf("expected distinct cross-process positions, both allocated %d", receipts[0].Seq)
	}
	for i, receipt := range receipts {
		if receipt.Seq < 5 {
			t.Errorf("child %d position %d is not above the existing maximum 4", i, receipt.Seq)
		}
	}
	payloads := [][]byte{sequencePNG(t, 0x3C), sequencePNG(t, 0xC3)}
	verifyPhotoSequence(t, dbPath, blobRoot, mementoID, receipts, payloads, existing, existingBytes)
}
