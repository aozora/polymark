package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aozora/polymark/internal/model"
)

func TestCreateSaveReopen(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	folder, err := st.CreateFolder("", "notes")
	if err != nil {
		t.Fatal(err)
	}
	rel, session, err := st.CreateDocument(folder, "Ciao", model.LangItalian)
	if err != nil {
		t.Fatal(err)
	}
	if rel != "notes/Ciao.md" {
		t.Fatalf("rel %s", rel)
	}
	end := session.Rich.Text().Len()
	session.Rich.Edit(func(e *modelEdit) {
		e.ReplaceString(end, end, "bold", modelAttr())
		e.Format(end, end+4, func(a *richtextAttr) { a.Weight = 700 })
	})
	if err := st.Save(session, session.SavedDigest()); err != nil {
		t.Fatal(err)
	}
	loaded, err := st.Load(rel)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Meta.Language != "it" || loaded.Meta.ID != session.ID {
		t.Fatalf("meta %+v", loaded.Meta)
	}
	if !strings.Contains(loaded.Markdown, "# Ciao") || !strings.Contains(loaded.Markdown, "**bold**") {
		t.Fatalf("markdown:\n%s", loaded.Markdown)
	}
	if len(loaded.Meta.Passages) == 0 {
		t.Fatal("no passages stored")
	}
	again := model.OpenMarkdown(rel, loaded.Markdown, loaded.Meta.ID, model.LangItalian, loaded.Meta.Passages)
	if again.Passages()[0].ID != loaded.Meta.Passages[0].ID {
		t.Fatalf("id %s want %s", again.Passages()[0].ID, loaded.Meta.Passages[0].ID)
	}
}

func TestTableFileIsNotRewritten(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	original := "| 語 | б |\n| --- | --- |\n| 漢 | 👨‍👩‍👧 |\n"
	rel := "table.md"
	session := model.OpenMarkdown(rel, original, "", model.LangJapanese, nil)
	if err := st.Save(session, ""); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf("saved %q", got)
	}
	loaded, err := st.Load(rel)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Markdown != original || loaded.Meta.Fidelity != string(model.FidelitySource) {
		t.Fatalf("loaded fidelity %s markdown %q", loaded.Meta.Fidelity, loaded.Markdown)
	}
}

func TestConflictDoesNotClobber(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, session, err := st.CreateDocument("", "Note", model.LangEnglish)
	if err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(dir, session.RelPath)
	if err := os.WriteFile(full, []byte("# external\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	session.Rich.Edit(func(e *modelEdit) {
		e.ReplaceString(0, 0, "X", modelAttr())
	})
	err = st.Save(session, session.SavedDigest())
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("err %v", err)
	}
	got, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "# external\n" {
		t.Fatalf("file became %q", got)
	}
}

func TestInterruptedWriteLeavesOriginal(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, session, err := st.CreateDocument("", "Note", model.LangEnglish)
	if err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(dir, session.RelPath)
	before, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	st.beforeRename = func(string) error { return errors.New("disk full") }
	session.Rich.Edit(func(e *modelEdit) {
		e.ReplaceString(0, 0, "Z", modelAttr())
	})
	if err := st.Save(session, session.SavedDigest()); err == nil {
		t.Fatal("save succeeded")
	}
	got, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(before) {
		t.Fatalf("original changed to %q", got)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".polymark-tmp-*"))
	if len(matches) != 0 {
		t.Fatalf("temp left behind: %v", matches)
	}
}

func TestSymlinkAndTraversal(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "link.md")); err != nil {
		t.Fatal(err)
	}
	session := model.OpenMarkdown("link.md", "changed\n", "", model.LangEnglish, nil)
	if err := st.Save(session, ""); !errors.Is(err, ErrSymlink) {
		t.Fatalf("symlink err %v", err)
	}
	got, _ := os.ReadFile(outside)
	if string(got) != "keep" {
		t.Fatalf("outside became %q", got)
	}
	if _, _, err := st.CreateDocument("..", "escape", model.LangEnglish); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("traversal err %v", err)
	}
	if _, err := st.CreateFolder("", ".."); !errors.Is(err, ErrName) {
		t.Fatalf("name err %v", err)
	}
	if _, _, err := st.CreateDocument("", "a/b", model.LangEnglish); !errors.Is(err, ErrName) {
		t.Fatalf("slash err %v", err)
	}
}

func TestRecoveryReopensUnsaved(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, session, err := st.CreateDocument("", "Draft", model.LangRussian)
	if err != nil {
		t.Fatal(err)
	}
	session.Rich.Edit(func(e *modelEdit) {
		e.ReplaceString(e.Text().Len(), e.Text().Len(), " привет", modelAttr())
	})
	session.SetSelection(1, 1)
	rec := Recovery{
		DocID:       session.ID,
		RelPath:     session.RelPath,
		Language:    string(session.Language),
		Mode:        string(session.Mode),
		Fidelity:    string(session.Fidelity),
		Markdown:    session.CommittedMarkdown(),
		Anchor:      1,
		Caret:       1,
		Passages:    session.Passages(),
		SavedDigest: session.SavedDigest(),
	}
	if err := st.WriteRecovery(rec); err != nil {
		t.Fatal(err)
	}
	// The process is gone. A new store reads the journal.
	st2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	recs, err := st2.Recoveries()
	if err != nil || len(recs) != 1 {
		t.Fatalf("recoveries %v %v", recs, err)
	}
	if !strings.Contains(recs[0].Markdown, "привет") || recs[0].Caret != 1 {
		t.Fatalf("recovery %+v", recs[0])
	}
	onDisk, err := os.ReadFile(filepath.Join(dir, session.RelPath))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(onDisk), "привет") {
		t.Fatal("unsaved edit was written to the document")
	}
}
