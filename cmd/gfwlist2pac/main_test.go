package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadInputFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	inPath := filepath.Join(tmpDir, "gfwlist.txt")
	want := "sample-data"
	if err := os.WriteFile(inPath, []byte(want), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	got, err := readInput(inPath, "")
	if err != nil {
		t.Fatalf("readInput returned error: %v", err)
	}

	if string(got) != want {
		t.Fatalf("readInput mismatch\nwant: %q\n got: %q", want, string(got))
	}
}
