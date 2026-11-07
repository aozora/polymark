package library

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/richtext"

	"github.com/aozora/polymark/internal/model"
)

func TestSwitchKeepsUnsavedAndRecoveryReopens(t *testing.T) {
	dir := t.TempDir()
	lib, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	first, err := lib.CreateDocument("One", model.LangEnglish)
	if err != nil {
		t.Fatal(err)
	}
	first.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(e.Text().Len(), e.Text().Len(), " unsaved", richtext.Attributes{})
	})
	if err := lib.Checkpoint(); err != nil {
		t.Fatal(err)
	}
	if _, err := lib.CreateDocument("Two", model.LangJapanese); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lib.Open[first.RelPath].CommittedMarkdown(), "unsaved") {
		t.Fatal("switching away dropped the unsaved edit")
	}
	lib2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := lib2.Open[first.RelPath]
	if got == nil || !strings.Contains(got.CommittedMarkdown(), "unsaved") || !got.Dirty() {
		md := ""
		if got != nil {
			md = got.CommittedMarkdown()
		}
		t.Fatalf("reopen dirty=%v markdown %q", got != nil && got.Dirty(), md)
	}
}
