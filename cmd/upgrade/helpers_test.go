package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// errReader fails after emitting some bytes, simulating an interrupted download.
type errReader struct {
	data []byte
	pos  int
}

func (r *errReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("simulated network error")
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func TestInstallBinaryReplacesAtomically(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "koff")
	if err := os.WriteFile(dest, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := installBinary(dest, strings.NewReader("NEWBINARY")); err != nil {
		t.Fatalf("installBinary: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "NEWBINARY" {
		t.Errorf("dest content = %q, want NEWBINARY", got)
	}
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("installed binary is not executable, mode = %v", info.Mode())
	}
	// No leftover temp files.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected only the final binary, found %d entries", len(entries))
	}
}

func TestInstallBinaryKeepsOldOnFailure(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "koff")
	if err := os.WriteFile(dest, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := installBinary(dest, &errReader{data: []byte("PARTIAL")})
	if err == nil {
		t.Fatal("expected error from failed download, got nil")
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "OLD" {
		t.Errorf("dest was modified on failure: %q, want OLD", got)
	}
	// Temp file must have been cleaned up, leaving only the original binary.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected only the original binary, found %d entries", len(entries))
	}
}

func TestParseTag(t *testing.T) {
	valid := []string{"v0.9.1", "v1.0.0", "v10.20.30"}
	for _, v := range valid {
		if _, err := parseTag(v); err != nil {
			t.Errorf("parseTag(%q) unexpected error: %v", v, err)
		}
	}
	invalid := []string{"", "v", "1.0.0", "latest", "vX.Y.Z", "vabc"}
	for _, v := range invalid {
		if _, err := parseTag(v); err == nil {
			t.Errorf("parseTag(%q) expected error, got nil", v)
		}
	}
}

func TestParseTagOrdering(t *testing.T) {
	older, _ := parseTag("v0.9.1")
	newer, _ := parseTag("v1.2.0")
	if !older.LessThan(*newer) {
		t.Errorf("expected v0.9.1 < v1.2.0")
	}
}
