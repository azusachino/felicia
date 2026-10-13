package publication

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image/jpeg"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const testVideoKey = "media/live-photo.MOV"

func syntheticMOV(t *testing.T) []byte {
	t.Helper()
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Fatalf("ffmpeg is required for the real synthetic MOV fixture: %v", err)
	}
	filename := filepath.Join(t.TempDir(), "fixture.mov")
	output, err := exec.Command(ffmpeg, "-nostdin", "-v", "error", "-f", "lavfi", "-i", "color=c=red:s=16x16:d=0.2", "-metadata", "comment=private-fixture", "-an", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-y", filename).CombinedOutput()
	if err != nil {
		t.Fatalf("create synthetic MOV fixture: %v: %s", err, output)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSanitizePublicVideoRejectsInvalidInputs(t *testing.T) {
	for name, input := range map[string][]byte{
		"empty":   nil,
		"corrupt": []byte("not a video"),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := SanitizePublicVideo(testVideoKey, bytes.NewReader(input)); err == nil {
				t.Fatal("SanitizePublicVideo succeeded for invalid input")
			}
		})
	}
	if _, err := SanitizePublicVideo(testVideoKey, errorReader{}); err == nil {
		t.Fatal("SanitizePublicVideo succeeded after source read failure")
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("synthetic read failure") }

type videoMapSource map[string][]byte

func (source videoMapSource) Open(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := source[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

type failingMediaWriter struct{ memoryArtifactWriter }

func (writer *failingMediaWriter) WriteMedia(key string, _ io.Reader) error {
	if strings.HasSuffix(key, ".mp4") {
		return errors.New("synthetic write failure")
	}
	return nil
}

func TestCompilePublishesSanitizedPlayableVideoDerivative(t *testing.T) {
	video := syntheticMOV(t)
	read := readModelWithPublishedPhoto("media/live-photo.jpg")
	memento := read.mementos[read.journeys[0].ID][0]
	photo := read.photos[memento.ID][0]
	photo.VideoKey = stringPtr(testVideoKey)
	writer := &memoryArtifactWriter{}
	photoBytes := mustJPEGBytes(t)

	report, err := (StaticCompiler{}).Compile(context.Background(), Input{}, read, videoMapSource{
		photo.ObjectKey: photoBytes,
		testVideoKey:    video,
	}, writer)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if report.Media != 2 {
		t.Fatalf("report.Media = %d, want photo and video derivative", report.Media)
	}
	publicKey := PublicVideoKey(testVideoKey)
	derivative, ok := writer.mediaBytes[publicKey]
	if !ok || bytes.Equal(video, derivative) {
		t.Fatalf("video derivative at %q missing or copied verbatim", publicKey)
	}

	path := "api/v1/journeys/" + read.journeys[0].ID.String() + "/mementos.json"
	var mementos []StaticMemento
	if err := json.Unmarshal(writer.json[path], &mementos); err != nil {
		t.Fatalf("decode mementos: %v", err)
	}
	if len(mementos) != 1 || len(mementos[0].Photos) != 1 || mementos[0].Photos[0].VideoKey == nil || *mementos[0].Photos[0].VideoKey != publicKey {
		t.Fatalf("published photo video key = %+v, want %q", mementos, publicKey)
	}

	filename := filepath.Join(t.TempDir(), "derivative.mp4")
	if err := os.WriteFile(filename, derivative, 0o600); err != nil {
		t.Fatal(err)
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Fatalf("ffprobe is required to verify video fixture playability: %v", err)
	}
	probe, err := exec.Command(ffprobe, "-v", "error", "-show_entries", "stream=codec_name,pix_fmt", "-of", "json", filename).CombinedOutput()
	if err != nil {
		t.Fatalf("ffprobe derivative: %v: %s", err, probe)
	}
	if !strings.Contains(string(probe), `"codec_name": "h264"`) || !strings.Contains(string(probe), `"pix_fmt": "yuv420p"`) {
		t.Fatalf("derivative is not browser-compatible H.264/yuv420p: %s", probe)
	}
	tags, err := exec.Command(ffprobe, "-v", "error", "-show_entries", "format_tags", "-of", "json", filename).CombinedOutput()
	if err != nil {
		t.Fatalf("ffprobe derivative metadata: %v: %s", err, tags)
	}
	if strings.Contains(string(tags), "private-fixture") {
		t.Fatalf("private source metadata survived in derivative: %s", tags)
	}
}

func TestCompileFailsOnLinkedVideoOpenAndWriteErrors(t *testing.T) {
	read := readModelWithPublishedPhoto("media/live-photo.jpg")
	memento := read.mementos[read.journeys[0].ID][0]
	photo := read.photos[memento.ID][0]
	photo.VideoKey = stringPtr(testVideoKey)
	photoBytes := mustJPEGBytes(t)

	t.Run("open error", func(t *testing.T) {
		writer := &memoryArtifactWriter{}
		_, err := (StaticCompiler{}).Compile(context.Background(), Input{}, read, videoMapSource{photo.ObjectKey: photoBytes}, writer)
		if err == nil || !strings.Contains(err.Error(), "open video") {
			t.Fatalf("Compile error = %v, want linked video open failure", err)
		}
	})
	t.Run("corrupt linked video", func(t *testing.T) {
		writer := &memoryArtifactWriter{}
		_, err := (StaticCompiler{}).Compile(context.Background(), Input{}, read, videoMapSource{photo.ObjectKey: photoBytes, testVideoKey: []byte("corrupt")}, writer)
		if err == nil || !strings.Contains(err.Error(), "sanitize video") {
			t.Fatalf("Compile error = %v, want linked video sanitization failure", err)
		}
	})
	t.Run("write error", func(t *testing.T) {
		writer := &failingMediaWriter{}
		_, err := (StaticCompiler{}).Compile(context.Background(), Input{}, read, videoMapSource{photo.ObjectKey: photoBytes, testVideoKey: syntheticMOV(t)}, writer)
		if err == nil || !strings.Contains(err.Error(), "write video") {
			t.Fatalf("Compile error = %v, want video artifact write failure", err)
		}
	})
}

func mustJPEGBytes(t *testing.T) []byte {
	t.Helper()
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, gradient(8, 8), nil); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}
