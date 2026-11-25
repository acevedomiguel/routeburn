package routeburn_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"

	routeburn "github.com/acevedomiguel/routeburn"
)

func TestWriteJSONMatchesMarshalJSONAndPreservesNulls(t *testing.T) {
	snap := routeburn.Snapshot{
		SchemaVersion: routeburn.SchemaVersion,
		GeneratedAt:   fixedTime(),
		Key:           &routeburn.KeyCredits{},
	}
	want, err := routeburn.MarshalJSON(snap, false)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	var out bytes.Buffer
	if err := routeburn.WriteJSON(&out, snap, false); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if !bytes.Equal(out.Bytes(), want) {
		t.Fatalf("WriteJSON output differs:\n%s\nwant:\n%s", out.Bytes(), want)
	}

	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded["schema_version"] != float64(2) {
		t.Errorf("schema_version = %v, want 2", decoded["schema_version"])
	}
	runway := decoded["runway"].(map[string]any)
	if value, ok := runway["days_remaining"]; !ok || value != nil {
		t.Errorf("days_remaining = %v, present %v; want explicit null", value, ok)
	}
	key := decoded["key"].(map[string]any)
	if value, ok := key["limit"]; !ok || value != nil {
		t.Errorf("limit = %v, present %v; want explicit null", value, ok)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

func TestWriteJSONReturnsWriterError(t *testing.T) {
	want := errors.New("synthetic write failure")
	err := routeburn.WriteJSON(failingWriter{err: want}, routeburn.Snapshot{}, false)
	if !errors.Is(err, want) {
		t.Fatalf("WriteJSON error = %v, want %v", err, want)
	}
}

func TestWriteJSONRejectsShortWrite(t *testing.T) {
	err := routeburn.WriteJSON(shortWriter{}, routeburn.Snapshot{}, false)
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("WriteJSON error = %v, want io.ErrShortWrite", err)
	}
}
