package photos

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

type photoMemory struct {
	mu        sync.Mutex
	id        uuid.UUID
	rows      []*domain.MementoPhoto
	writeErr  error
	committed bool
	lookupErr error
}

func (m *photoMemory) GetMemento(_ context.Context, id uuid.UUID) (*domain.Memento, error) {
	if id != m.id {
		return nil, domain.ErrNotFound
	}
	return &domain.Memento{ID: id}, nil
}
func (m *photoMemory) GetPhoto(_ context.Context, id uuid.UUID) (*domain.MementoPhoto, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lookupErr != nil {
		return nil, m.lookupErr
	}
	for _, row := range m.rows {
		if row.ID == id {
			return row, nil
		}
	}
	return nil, domain.ErrNotFound
}

// CreatePhotoWithNextSequence mirrors the provider seam: allocation is
// computed from the rows under a lock and the insert never skips on conflict.
func (m *photoMemory) CreatePhotoWithNextSequence(_ context.Context, photo *domain.MementoPhoto) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.writeErr != nil {
		if m.committed {
			seq := 0
			for _, row := range m.rows {
				if row.Seq >= seq {
					seq = row.Seq + 1
				}
			}
			stored := *photo
			stored.Seq = seq
			m.rows = append(m.rows, &stored)
		}
		return 0, m.writeErr
	}
	seq := 0
	for _, row := range m.rows {
		if row.ID == photo.ID {
			return 0, errors.New("photo identity conflict")
		}
		if row.Seq >= seq {
			seq = row.Seq + 1
		}
	}
	stored := *photo
	stored.Seq = seq
	m.rows = append(m.rows, &stored)
	return seq, nil
}

// photoBlob is a concurrency-safe in-memory double for the blob port.
type photoBlob struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newPhotoBlob() *photoBlob { return &photoBlob{data: map[string][]byte{}} }

func (b *photoBlob) Delete(_ context.Context, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.data, key)
	return nil
}
func (b *photoBlob) Put(_ context.Context, key string, value []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data[key] = bytes.Clone(value)
	return nil
}
func (b *photoBlob) Open(_ context.Context, key string) (io.ReadCloser, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return io.NopCloser(bytes.NewReader(b.data[key])), nil
}

func TestUploadSerializesPhotoSequenceAndPreservesExistingIdentity(t *testing.T) {
	repo := &photoMemory{id: uuid.New()}
	original := &domain.MementoPhoto{ID: uuid.New(), MementoID: repo.id, Seq: 4, ObjectKey: "existing/original.jpg", ContentHash: "original identity"}
	repo.rows = []*domain.MementoPhoto{original}
	blobs := newPhotoBlob()
	blobs.data[original.ObjectKey] = []byte("unchanged original")
	upload := New(repo, blobs)
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewGray(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if _, err := upload.Upload(context.Background(), repo.id, data.Bytes()); err != nil {
				t.Error(err)
			}
		}()
	}
	workers.Wait()
	if len(repo.rows) != 9 || len(blobs.data) != 9 {
		t.Fatalf("rows=%d originals=%d", len(repo.rows), len(blobs.data))
	}
	if repo.rows[0] != original || string(blobs.data[original.ObjectKey]) != "unchanged original" {
		t.Fatal("existing identity or bytes changed")
	}
	for i, row := range repo.rows[1:] {
		if row.Seq != i+5 || row.MementoID != repo.id || row.SourceRef != nil || !bytes.Equal(blobs.data[row.ObjectKey], data.Bytes()) {
			t.Fatalf("row identity %+v", row)
		}
	}
}

func TestUploadCompensationDoesNotDeleteCommittedOrExistingOriginals(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		committed bool
		lookupErr error
		originals int
	}{
		{"rejected metadata", false, nil, 1},
		{"uncertain committed write", true, nil, 2},
		{"unknown repository outcome", false, errors.New("repository unavailable"), 2},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var data bytes.Buffer
			if err := png.Encode(&data, image.NewGray(image.Rect(0, 0, 8, 8))); err != nil {
				t.Fatal(err)
			}
			existing := &domain.MementoPhoto{ID: uuid.New(), ObjectKey: "existing/original.png", Seq: 4}
			repo := &photoMemory{id: uuid.New(), rows: []*domain.MementoPhoto{existing}, writeErr: context.Canceled, committed: scenario.committed, lookupErr: scenario.lookupErr}
			blobs := newPhotoBlob()
			blobs.data[existing.ObjectKey] = bytes.Clone(data.Bytes())
			if _, err := New(repo, blobs).Upload(context.Background(), repo.id, data.Bytes()); !errors.Is(err, context.Canceled) {
				t.Fatalf("write error: %v", err)
			}
			if len(blobs.data) != scenario.originals || !bytes.Equal(blobs.data[existing.ObjectKey], data.Bytes()) {
				t.Fatalf("unsafe compensation: %d originals", len(blobs.data))
			}
			for _, photo := range repo.rows {
				if _, exists := blobs.data[photo.ObjectKey]; !exists {
					t.Fatal("committed row lost its original")
				}
			}
		})
	}
}

func TestImageFormatPreservesSharedUploadLimits(t *testing.T) {
	var jpegBytes, pngBytes bytes.Buffer
	if err := jpeg.Encode(&jpegBytes, image.NewGray(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&pngBytes, image.NewGray(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	webp, err := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	if err != nil {
		t.Fatal(err)
	}
	for format, data := range map[string][]byte{"jpeg": jpegBytes.Bytes(), "png": pngBytes.Bytes(), "webp": webp} {
		t.Run(format, func(t *testing.T) {
			got, err := ImageFormat(data)
			if err != nil || got != format {
				t.Fatalf("format=%q error=%v", got, err)
			}
		})
	}
	for _, data := range [][]byte{[]byte("<svg><script/></svg>"), []byte("GIF89a"), []byte("\x00\x00\x00\x18ftypheic"), []byte("not an image")} {
		var invalid *InvalidImageError
		if _, err := ImageFormat(data); !errors.As(err, &invalid) {
			t.Fatalf("accepted invalid image: %v", err)
		}
	}
	if _, err := ImageFormat(make([]byte, MaxUploadBytes+1)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("size limit: %v", err)
	}
	for _, dimensions := range [][2]uint32{{50_001, 1}, {10_000, 10_000}} {
		data := bytes.Clone(pngBytes.Bytes())
		binary.BigEndian.PutUint32(data[16:20], dimensions[0])
		binary.BigEndian.PutUint32(data[20:24], dimensions[1])
		binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
		if _, err := ImageFormat(data); err == nil || err.Error() != "image dimensions are too large" {
			t.Fatalf("dimensions %v: %v", dimensions, err)
		}
	}
}
