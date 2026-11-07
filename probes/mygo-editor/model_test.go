package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/richtext"
)

func TestMarkdownRoundTripKeepsScriptsAndMarks(t *testing.T) {
	source := strings.Join([]string{
		"# 見出し Heading Заголовок",
		"",
		"Latin, Cyrillic привет, and 日本語.",
		"",
		"か\u3099 combining beside が.",
		"",
		"Family 👨‍👩‍👧‍👦 and a👨‍👩‍👧‍👦b.",
		"",
		"**bold** and *italic* and a [link](https://example.test/путь).",
		"",
		"- 一",
		"- two",
		"- три",
	}, "\n")
	text := richtext.ParseMarkdown(source)
	again := richtext.ParseMarkdown(text.Markdown())
	if text.Markdown() != again.Markdown() {
		t.Fatalf("second export changed:\n%s", again.Markdown())
	}
	plain := text.String()
	for _, want := range []string{"見出し", "привет", "日本語", "か\u3099", "が", "👨‍👩‍👧‍👦", "bold", "italic", "https://example.test/путь"} {
		if !strings.Contains(plain, want) && !strings.Contains(text.Markdown(), want) {
			t.Fatalf("missing %q\nplain: %s\nmd:\n%s", want, plain, text.Markdown())
		}
	}
	if text.ParagraphStyleAt(0).Heading != 1 {
		t.Fatalf("heading level %d", text.ParagraphStyleAt(0).Heading)
	}
}

func TestTablesBecomeTabSeparatedParagraphs(t *testing.T) {
	source := "| 語 | б |\n| --- | --- |\n| 漢 | 👨‍👩‍👧 |"
	text := richtext.ParseMarkdown(source)
	plain := text.String()
	if !strings.Contains(plain, "語\tб") || !strings.Contains(plain, "漢\t👨‍👩‍👧") {
		t.Fatalf("table cells were not kept as tab-separated text: %q", plain)
	}
	exported := text.Markdown()
	if strings.Contains(exported, "| --- |") {
		t.Fatalf("export unexpectedly restored a pipe table:\n%s", exported)
	}
	t.Logf("table export:\n%s", exported)
}

func TestExplicitEditUndoesTextAndSelection(t *testing.T) {
	doc := richtext.NewDocument(richtext.ParseMarkdown("before привет after"))
	doc.SetSelection(0, 0)
	anchor, caret := doc.Selection()
	version := doc.Version()
	doc.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 0, "雪", richtext.Attributes{})
	})
	if doc.Version() == version {
		t.Fatal("edit did not bump the document version")
	}
	if !strings.HasPrefix(doc.Text().String(), "雪") {
		t.Fatalf("text %q", doc.Text().String())
	}
	if !doc.CanUndo() {
		t.Fatal("explicit edit is not undoable")
	}
	doc.Undo()
	if got := doc.Text().String(); !strings.Contains(got, "before привет after") {
		t.Fatalf("undo text %q", got)
	}
	gotA, gotC := doc.Selection()
	if gotA != anchor || gotC != caret {
		t.Fatalf("selection after undo (%d,%d), before edit (%d,%d)", gotA, gotC, anchor, caret)
	}
}

func TestSetTextClearsHistory(t *testing.T) {
	doc := richtext.NewDocument(richtext.NewText("keep"))
	end := doc.Text().Len()
	doc.Edit(func(e *richtext.Edit) {
		e.ReplaceString(end, end, " me", richtext.Attributes{})
	})
	if !doc.CanUndo() {
		t.Fatal("expected an undo step")
	}
	doc.SetText(richtext.ParseMarkdown("replaced 日本語"))
	if doc.CanUndo() || doc.CanRedo() {
		t.Fatal("SetText kept history")
	}
	anchor, caret := doc.Selection()
	if anchor != caret || caret != doc.Text().Len() {
		t.Fatalf("selection after SetText (%d,%d), len %d", anchor, caret, doc.Text().Len())
	}
}

func TestParagraphIDsSurviveUndo(t *testing.T) {
	doc := richtext.NewDocument(richtext.ParseMarkdown("one\n\ntwo"))
	n := 0
	doc.IdentifyParagraphs(func() string {
		n++
		return "p" + string(rune('0'+n))
	})
	first := paragraphIDs(doc.Text())
	if len(first) < 2 || first[0] == "" || first[1] == "" {
		t.Fatalf("ids %q", first)
	}
	doc.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "X", richtext.Attributes{})
	})
	doc.Undo()
	if got := paragraphIDs(doc.Text()); strings.Join(got, ",") != strings.Join(first, ",") {
		t.Fatalf("ids after undo %q, want %q", got, first)
	}
}

func paragraphIDs(text richtext.Text) []string {
	var ids []string
	for _, p := range text.Paragraphs() {
		ids = append(ids, p.Style.ID)
	}
	return ids
}
