package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui/richtext"

	"github.com/aozora/polymark/internal/model"
)

func TestTargetSwitchKeepsEditsAndRecoveryKeepsProtection(t *testing.T) {
	dir := t.TempDir()
	lib, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	src, err := lib.CreateDocument("Hello", model.LangEnglish)
	if err != nil {
		t.Fatal(err)
	}
	src.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(e.Text().Len(), e.Text().Len(), "\n\nSecond", richtext.Attributes{})
	})
	if err := lib.SaveCurrent(); err != nil {
		t.Fatal(err)
	}
	it, err := lib.CreateTarget(model.LangItalian)
	if err != nil {
		t.Fatal(err)
	}
	ru, err := lib.CreateTarget(model.LangRussian)
	if err != nil {
		t.Fatal(err)
	}
	it.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "ciao", richtext.Attributes{})
	})
	if _, err := lib.ShowTarget(model.LangRussian); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lib.Open[it.RelPath].Rich.Text().String(), "ciao") {
		t.Fatal("Italian edit was dropped while showing Russian")
	}
	if _, err := lib.ShowTarget(model.LangItalian); err != nil {
		t.Fatal(err)
	}
	lib.Links.Observe(it, time.Unix(1, 0))
	if !strings.Contains(lib.Links.Status(src.RelPath, src.Passages()[0].ID, model.LangItalian), "protected") {
		t.Fatalf("status %s", lib.Links.Status(src.RelPath, src.Passages()[0].ID, model.LangItalian))
	}
	if err := lib.Checkpoint(); err != nil {
		t.Fatal(err)
	}
	lib2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if lib2.Current != src.RelPath {
		t.Fatalf("reopen current %s, want source %s", lib2.Current, src.RelPath)
	}
	if _, err := lib2.Focus(src.RelPath); err != nil {
		t.Fatal(err)
	}
	if _, err := lib2.ShowTarget(model.LangItalian); err != nil {
		t.Fatal(err)
	}
	got := lib2.TargetSession()
	if got == nil || !strings.Contains(got.Rich.Text().String(), "ciao") || !got.Dirty() {
		t.Fatal("recovery lost the Italian edit")
	}
	if !strings.Contains(lib2.Links.Status(src.RelPath, src.Passages()[0].ID, model.LangItalian), "protected") {
		t.Fatalf("reopen status %s", lib2.Links.Status(src.RelPath, src.Passages()[0].ID, model.LangItalian))
	}
	_ = ru
}

func TestSchema1ProjectStillOpens(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "polymark.json"), []byte("{\n  \"schema\": 1,\n  \"id\": \"proj\",\n  \"name\": \"Old\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	table := "| a | b |\n| --- | --- |\n| 1 | 2 |\n"
	if err := os.WriteFile(filepath.Join(dir, "table.md"), []byte(table), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "table.md.polymark.json"), []byte("{\n  \"schema\": 1,\n  \"id\": \"doc\",\n  \"language\": \"en\",\n  \"fidelity\": \"source\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lib, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s, err := lib.Focus("table.md")
	if err != nil {
		t.Fatal(err)
	}
	if s.Fidelity != model.FidelitySource || s.CommittedMarkdown() != table {
		t.Fatalf("fidelity %s markdown %q", s.Fidelity, s.CommittedMarkdown())
	}
	if err := s.SwitchMode(model.ModeRich); err == nil {
		t.Fatal("old table file became rich text")
	}
	if _, err := lib.CreateTarget(model.LangItalian); err == nil {
		t.Fatal("created a translation target for a table file")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".it.md") {
			t.Fatal("table link wrote a target file")
		}
	}
}

func TestSaveReopenKeepsTargetPassageIds(t *testing.T) {
	dir := t.TempDir()
	lib, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	src, err := lib.CreateDocument("Hello", model.LangEnglish)
	if err != nil {
		t.Fatal(err)
	}
	src.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(e.Text().Len(), e.Text().Len(), "\n\nSecond", richtext.Attributes{})
	})
	if err := lib.SaveCurrent(); err != nil {
		t.Fatal(err)
	}
	it, err := lib.CreateTarget(model.LangItalian)
	if err != nil {
		t.Fatal(err)
	}
	ids := src.Passages()
	if len(ids) < 2 || len(it.Passages()) != len(ids) {
		t.Fatalf("source %d target %d", len(ids), len(it.Passages()))
	}
	it.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "ciao", richtext.Attributes{})
	})
	lib.Links.Observe(it, time.Unix(1, 0))
	if err := lib.SaveEditing(); err != nil {
		t.Fatal(err)
	}
	lib2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lib2.Focus(src.RelPath); err != nil {
		t.Fatal(err)
	}
	got, err := lib2.ShowTarget(model.LangItalian)
	if err != nil || got == nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.CommittedMarkdown(), "ciao") {
		t.Fatalf("reopen markdown %q", got.CommittedMarkdown())
	}
	if len(got.Passages()) != len(ids) {
		t.Fatalf("passages %d want %d\n%s", len(got.Passages()), len(ids), got.CommittedMarkdown())
	}
	for i := range ids {
		if got.Passages()[i].ID != ids[i].ID {
			t.Fatalf("passage %d id %s want %s", i, got.Passages()[i].ID, ids[i].ID)
		}
	}
	if got := lib2.Links.Status(src.RelPath, ids[0].ID, model.LangItalian); got != "protected, outdated" {
		t.Fatalf("saved status %q", got)
	}
	if got := lib2.Links.Status(src.RelPath, ids[1].ID, model.LangItalian); got != "outdated" {
		t.Fatalf("untouched passage %q", got)
	}
}
