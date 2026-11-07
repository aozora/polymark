package appui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"

	"github.com/aozora/polymark/internal/library"
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
	before := session.CommittedMarkdown()
	version := session.Rich.Version()
	tt.Compose("に", 1)
	tt.Frame()
	if session.CommittedMarkdown() != before || session.Rich.Version() != version {
		t.Fatalf("pre-edit committed: %q version %d -> %d", session.CommittedMarkdown(), version, session.Rich.Version())
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

func richtextEditor(c *ui.Context, session *model.Session) {
	richtext.Editor(c, session.Rich).Fill()
}
