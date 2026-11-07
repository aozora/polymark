package model

import (
	"strings"
	"testing"
)

func TestRoundTripRequiredFormatting(t *testing.T) {
	src := strings.Join([]string{
		"# 見出し Heading Заголовок",
		"",
		"Latin, привет, and 日本語 with か\u3099 and が and 👨‍👩‍👧‍👦.",
		"",
		"**bold** and *italic* and a [link](https://example.test/путь).",
		"",
		"- 一",
		"- two",
		"- три",
	}, "\n")
	s := OpenMarkdown("notes/a.md", src, "doc1", LangJapanese, nil)
	if s.Fidelity != FidelityRich || s.Mode != ModeRich {
		t.Fatalf("fidelity %s mode %s", s.Fidelity, s.Mode)
	}
	again := OpenMarkdown("notes/a.md", s.CommittedMarkdown(), s.ID, s.Language, s.Passages())
	if !strings.Contains(again.CommittedMarkdown(), "# 見出し") ||
		!strings.Contains(again.CommittedMarkdown(), "**bold**") ||
		!strings.Contains(again.CommittedMarkdown(), "*italic*") ||
		!strings.Contains(again.CommittedMarkdown(), "https://example.test/путь") ||
		!strings.Contains(again.CommittedMarkdown(), "- 一") {
		t.Fatalf("export lost formatting:\n%s", again.CommittedMarkdown())
	}
	if !strings.Contains(again.Rich.Text().String(), "か\u3099") || !strings.Contains(again.Rich.Text().String(), "👨‍👩‍👧‍👦") {
		t.Fatalf("plain %q", again.Rich.Text().String())
	}
	if len(s.Passages()) == 0 || again.Passages()[0].ID != s.Passages()[0].ID {
		t.Fatalf("passage ids %v then %v", s.Passages(), again.Passages())
	}
}

func TestPipeTableStaysOutOfRichText(t *testing.T) {
	src := "| 語 | б |\n| --- | --- |\n| 漢 | 👨‍👩‍👧 |\n"
	s := OpenMarkdown("t.md", src, "", LangEnglish, nil)
	if s.Fidelity != FidelitySource || s.Mode != ModeSource {
		t.Fatalf("got fidelity %s mode %s", s.Fidelity, s.Mode)
	}
	if s.Rich != nil {
		t.Fatal("table was parsed into the rich editor")
	}
	if s.CommittedMarkdown() != src {
		t.Fatalf("source changed:\n%q", s.CommittedMarkdown())
	}
	if err := s.SwitchMode(ModeRich); err == nil {
		t.Fatal("rich mode accepted a table")
	}
}

func TestEditUndoRestoresTextAndSelection(t *testing.T) {
	s := OpenMarkdown("a.md", "before привет after\n", "", LangRussian, nil)
	s.SetSelection(0, 0)
	s.Rich.Edit(func(e *richtextEdit) {})
	anchor, caret := s.Selection()
	s.Rich.Edit(func(e *richtextEdit) {
		e.ReplaceString(0, 0, "雪", attr())
	})
	if !strings.HasPrefix(s.Rich.Text().String(), "雪") {
		t.Fatalf("text %q", s.Rich.Text().String())
	}
	s.Rich.Undo()
	if !strings.Contains(s.Rich.Text().String(), "before привет after") {
		t.Fatalf("undo text %q", s.Rich.Text().String())
	}
	gotA, gotC := s.Selection()
	if gotA != anchor || gotC != caret {
		t.Fatalf("selection (%d,%d), want (%d,%d)", gotA, gotC, anchor, caret)
	}
}

func TestModeSwitchKeepsUndoUntilSourceChanges(t *testing.T) {
	s := OpenMarkdown("a.md", "hello\n", "", LangEnglish, nil)
	s.Rich.Edit(func(e *richtextEdit) {
		e.ReplaceString(e.Text().Len(), e.Text().Len(), "!", attr())
	})
	if err := s.SwitchMode(ModeSource); err != nil {
		t.Fatal(err)
	}
	if err := s.SwitchMode(ModeRich); err != nil {
		t.Fatal(err)
	}
	if !s.Rich.CanUndo() {
		t.Fatal("returning without a source edit cleared rich undo")
	}
	s.Rich.Undo()
	if strings.Contains(s.CommittedMarkdown(), "!") {
		t.Fatalf("undo lost the pre-switch edit:\n%s", s.CommittedMarkdown())
	}

	if err := s.SwitchMode(ModeSource); err != nil {
		t.Fatal(err)
	}
	s.NoteSource(s.Source+"\n\n**added**\n", 0, 0)
	if err := s.SwitchMode(ModeRich); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s.CommittedMarkdown(), "**added**") {
		t.Fatalf("source edit missing:\n%s", s.CommittedMarkdown())
	}
	if !s.Rich.CanUndo() {
		t.Fatal("applying source used a history reset instead of an undo step")
	}
	s.Rich.Undo()
	if strings.Contains(s.CommittedMarkdown(), "added") {
		t.Fatalf("undo did not restore pre-source text:\n%s", s.CommittedMarkdown())
	}
}

func TestSourceTableBlocksReturnToRich(t *testing.T) {
	s := OpenMarkdown("a.md", "hello\n", "", LangItalian, nil)
	if err := s.SwitchMode(ModeSource); err != nil {
		t.Fatal(err)
	}
	s.NoteSource("| a | b |\n| --- | --- |\n| 1 | 2 |\n", 0, 0)
	err := s.SwitchMode(ModeRich)
	if err == nil {
		t.Fatal("table in source was accepted")
	}
	if s.Mode != ModeSource {
		t.Fatalf("mode %s", s.Mode)
	}
	if !strings.Contains(s.CommittedMarkdown(), "| --- |") {
		t.Fatalf("table source lost:\n%s", s.CommittedMarkdown())
	}
}

func TestTwoEditsUndoOneStep(t *testing.T) {
	s := OpenMarkdown("a.md", "one\n", "", LangEnglish, nil)
	s.Rich.Edit(func(e *richtextEdit) {
		e.ReplaceString(e.Text().Len(), e.Text().Len(), "A", attr())
	})
	s.Rich.Edit(func(e *richtextEdit) {
		e.ReplaceString(e.Text().Len(), e.Text().Len(), "B", attr())
	})
	s.Rich.Undo()
	plain := s.Rich.Text().String()
	if !strings.Contains(plain, "A") || strings.Contains(plain, "B") {
		t.Fatalf("one undo left %q", plain)
	}
}
