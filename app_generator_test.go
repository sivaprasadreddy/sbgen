package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateUniqueZipFileUsesNextAvailableSuffix(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"demo.zip", "demo-1.zip"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("existing"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	file, filePath, err := createUniqueZipFile(directory, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(directory, "demo-2.zip")
	if filePath != want {
		t.Fatalf("createUniqueZipFile() path = %q, want %q", filePath, want)
	}
	for _, name := range []string{"demo.zip", "demo-1.zip"} {
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "existing" {
			t.Fatalf("existing archive %q was overwritten", name)
		}
	}
}
