package appui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"

	"github.com/aozora/polymark/internal/library"
	"github.com/aozora/polymark/internal/link"
	"github.com/aozora/polymark/internal/model"
	"github.com/aozora/polymark/internal/theme"
)

func TestWindowEditsAndKeepsPreEditOut(t *testing.T) {
	dir := t.TempDir()
	lib, err := library.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	app := New(lib, theme.Tokens{})
	tt := ui.NewTester(app.View, 1100, 720)
	app.Name = "Note"
	if err := tt.Click("New document"); err != nil {
		t.Fatal(err)
	}
	s := lib.CurrentSession()
	if s == nil || s.Mode != model.ModeRich {
		t.Fatal("document was not opened in rich text")
	}
	if err := tt.Click("Markdown"); err != nil {
		t.Fatal(err)
	}
	if s.Mode != model.ModeSource || !strings.Contains(s.Source, "# Note") {
		t.Fatalf("source mode %s %q", s.Mode, s.Source)
	}
	if err := tt.Click("Rich text"); err != nil {
		t.Fatal(err)
	}
	if s.Mode != model.ModeRich {
		t.Fatalf("mode %s", s.Mode)
	}
}

func TestEditorPreEditIsNotCommitted(t *testing.T) {
	dir := t.TempDir()
	lib, err := library.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	session, err := lib.CreateDocument("Note", model.LangJapanese)
	if err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		richtextEditor(c, session)
	}, 640, 400)
	tt.Press(320, 300)
	tt.Release(320, 300)
	if active, _ := tt.TextInputActive(); !active {
		t.Fatal("editor did not take focus")
	}
	if _, err := lib.CreateTarget(model.LangItalian); err != nil {
		t.Fatal(err)
	}
	rev := lib.Links.Group(session.RelPath).Passages[0].SourceRevision
	before := session.CommittedMarkdown()
	version := session.Rich.Version()
	tt.Compose("に", 1)
	tt.Frame()
	lib.Links.Observe(session, tt.Now())
	if session.CommittedMarkdown() != before || session.Rich.Version() != version {
		t.Fatalf("pre-edit committed: %q version %d -> %d", session.CommittedMarkdown(), version, session.Rich.Version())
	}
	if got := lib.Links.Group(session.RelPath).Passages[0].SourceRevision; got != rev {
		t.Fatalf("pre-edit changed source revision %d -> %d", rev, got)
	}
	caret, ok := tt.TextCaret()
	if !ok || caret.H <= 0 {
		t.Fatalf("caret %+v ok=%v", caret, ok)
	}
	tt.Compose("", 0)
	tt.Type("x")
	tt.Frame()
	if strings.Contains(session.CommittedMarkdown(), "に") {
		t.Fatalf("cancelled pre-edit stored: %q", session.CommittedMarkdown())
	}
	if !strings.Contains(session.Rich.Text().String(), "x") {
		t.Fatalf("typed text missing from %q", session.Rich.Text().String())
	}
	boxes := session.Rich.VisibleParagraphs()
	if len(boxes) == 0 || boxes[0].Height <= 0 {
		t.Fatalf("boxes %+v", boxes)
	}
}

func TestLinkedPanesStayHonest(t *testing.T) {
	dir := t.TempDir()
	lib, err := library.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	app := New(lib, theme.Tokens{})
	tt := ui.NewTester(app.View, 1280, 800)
	app.Name = "Hello"
	if err := tt.Click("New document"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Translation is not configured. No text is sent.") {
		t.Fatal("window did not say that translation is not configured")
	}
	if tt.HasText("FAKE") {
		t.Fatal("window showed a fixture translation")
	}
	src := lib.CurrentSession()
	src.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(e.Text().Len(), e.Text().Len(), "\n\nSecond", richtext.Attributes{})
	})
	if err := tt.Click("Show Italian"); err != nil {
		t.Fatal(err)
	}
	it := lib.TargetSession()
	if it == nil || it.Language != model.LangItalian {
		t.Fatal("Italian target was not opened")
	}
	if strings.Contains(it.CommittedMarkdown(), "Second") {
		t.Fatalf("target copied the source:\n%s", it.CommittedMarkdown())
	}
	if err := tt.Click("Italian"); err != nil {
		t.Fatal(err)
	}
	if src.Language != model.LangEnglish {
		t.Fatalf("source language became %s", src.Language)
	}
	ids := src.Passages()
	if len(ids) < 2 {
		t.Fatal(ids)
	}
	app.align()
	start, ok := link.PassageStart(src, ids[1].ID)
	if !ok {
		t.Fatal("second passage")
	}
	src.SetSelection(start, start)
	app.align()
	_, caret := it.Selection()
	gotID, _, ok := link.PassageAt(it, caret)
	if !ok || gotID != ids[1].ID {
		t.Fatalf("target caret passage %q want %q", gotID, ids[1].ID)
	}
	app.Sync = false
	src.SetSelection(0, 0)
	app.align()
	_, caret = it.Selection()
	gotID, _, _ = link.PassageAt(it, caret)
	if gotID != ids[1].ID {
		t.Fatal("turning scroll together off moved the target")
	}
	it.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "manuale", richtext.Attributes{})
	})
	app.checkpoint(it)
	it.SetSelection(0, 0)
	app.Replace = "ciao"
	if err := tt.Click("Replace passage"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(it.Rich.Text().String(), "ciao") || strings.Contains(it.Rich.Text().String(), "manuale") {
		t.Fatalf("replace text %q", it.Rich.Text().String())
	}
	if err := tt.Click("Undo replace"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(it.Rich.Text().String(), "ciao") || !strings.Contains(it.Rich.Text().String(), "manuale") {
		t.Fatalf("undo text %q", it.Rich.Text().String())
	}
	if got := lib.Links.Status(src.RelPath, ids[0].ID, model.LangItalian); got != "protected, outdated" {
		t.Fatalf("status %q", got)
	}
	if err := tt.Click("Show Russian"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lib.Open[it.RelPath].Rich.Text().String(), "manuale") {
		t.Fatal("switching target dropped the Italian edit")
	}
	if err := tt.Click("Show Italian"); err != nil {
		t.Fatal(err)
	}
	if lib.TargetSession() != lib.Open[it.RelPath] {
		t.Fatal("Italian target was reloaded over the edit")
	}
	src.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "Q", richtext.Attributes{})
	})
	app.checkpoint(src)
	if !strings.Contains(lib.Links.Status(src.RelPath, ids[0].ID, model.LangItalian), "outdated") {
		t.Fatal(lib.Links.Status(src.RelPath, ids[0].ID, model.LangItalian))
	}
	if !tt.HasText("Translation is not configured. No text is sent.") || tt.HasText("FAKE") {
		t.Fatal("configured state changed")
	}
}

func richtextEditor(c *ui.Context, session *model.Session) {
	richtext.Editor(c, session.Rich).Fill()
}
