package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

type notes struct {
	doc *richtext.Document
}

func (n *notes) view(c *ui.Context) {
	richtext.Editor(c, n.doc).Fill()
}

func TestEditorFormatsAndSimulatedComposition(t *testing.T) {
	doc := richtext.NewDocument(richtext.ParseMarkdown("hello\n\nworld"))
	app := &notes{doc: doc}
	tt := ui.NewTester(app.view, 640, 480)
	// The editor's glyphs are not a separate element named "hello".
	// A press in the text focuses it, as MyGo's own rich-text tests do.
	tt.Press(320, 420)
	tt.Release(320, 420)
	doc.SetSelection(0, 5)
	tt.Frame()
	doc.ToggleBold()
	doc.SetHeading(2)
	tt.Frame()
	exported := doc.Text().Markdown()
	if !strings.Contains(exported, "##") || !strings.Contains(exported, "**hello**") {
		t.Fatalf("formatted export:\n%s", exported)
	}

	doc.SetSelection(doc.Text().Len(), doc.Text().Len())
	tt.Frame()
	before := doc.Text().String()
	version := doc.Version()
	tt.Compose("に", 1)
	tt.Frame()
	during := doc.Text().String()
	t.Logf("during compose text=%q version %d -> %d", during, version, doc.Version())
	if strings.Contains(during, "に") {
		t.Fatalf("pre-edit was committed into the document: %q", during)
	}
	if during != before {
		t.Fatalf("pre-edit changed committed text from %q to %q", before, during)
	}
	caret, ok := tt.TextCaret()
	t.Logf("caret during compose %+v ok=%v", caret, ok)
	if !ok {
		t.Fatal("no caret geometry while composing")
	}
	if caret.H <= 0 {
		t.Fatalf("caret box has no height: %+v", caret)
	}

	tt.Compose("", 0)
	tt.Frame()
	if got := doc.Text().String(); got != before {
		t.Fatalf("cancel left %q", got)
	}

	tt.Compose("に", 1)
	tt.Type("に")
	tt.Frame()
	if !strings.Contains(doc.Text().String(), "に") {
		t.Fatalf("commit missing: %q", doc.Text().String())
	}
	if !doc.CanUndo() {
		t.Fatal("commit was not undoable")
	}
	doc.Undo()
	if got := doc.Text().String(); got != before {
		t.Fatalf("undo after commit left %q, want %q", got, before)
	}
}

func TestThemeSwitchKeepsTheDocument(t *testing.T) {
	doc := richtext.NewDocument(richtext.ParseMarkdown("**stay** 日本語"))
	app := &notes{doc: doc}
	tt := ui.NewTester(app.view, 480, 200)
	before := doc.Text().Markdown()
	anchor, caret := doc.Selection()
	tt.SetDark(true)
	tt.Frame()
	tt.SetDark(false)
	tt.Frame()
	if doc.Text().Markdown() != before {
		t.Fatalf("theme switch changed markdown:\n%s", doc.Text().Markdown())
	}
	gotA, gotC := doc.Selection()
	if gotA != anchor || gotC != caret {
		t.Fatalf("selection (%d,%d), want (%d,%d)", gotA, gotC, anchor, caret)
	}
}
