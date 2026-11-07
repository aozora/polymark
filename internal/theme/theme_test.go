package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRejectsScriptsAndUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "theme.json")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(script); err == nil {
		t.Fatal("script was accepted")
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{\"accent\":\"red\",\"exec\":\"rm\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(bad); err == nil {
		t.Fatal("unknown field was accepted")
	}
	ok := filepath.Join(dir, "ok.json")
	if err := os.WriteFile(ok, []byte("{\"accent\":\"#C4553A\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(ok)
	if err != nil || got.Accent != "#C4553A" {
		t.Fatalf("%+v %v", got, err)
	}
}
