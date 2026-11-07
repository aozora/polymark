package link

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui/richtext"

	"github.com/aozora/polymark/internal/model"
)

func linked(t *testing.T) (*Book, *model.Session, *model.Session) {
	t.Helper()
	src := model.OpenMarkdown("notes/Hello.md", "Hello\n\nWorld\n", "src", model.LangEnglish, nil)
	b := New()
	b.SetFocus(src.RelPath)
	b.SetVisible(model.LangItalian)
	rel, md, passages, err := PlanTarget(src, model.LangItalian)
	if err != nil {
		t.Fatal(err)
	}
	tgt := model.OpenMarkdown(rel, md, "tgt", model.LangItalian, passages)
	if err := b.AdoptTarget(src, tgt); err != nil {
		t.Fatal(err)
	}
	return b, src, tgt
}

func TestLinkStartsOutdatedAndObserveIsStable(t *testing.T) {
	b, src, tgt := linked(t)
	id := src.Passages()[0].ID
	if got := b.Status(src.RelPath, id, model.LangItalian); got != "outdated" {
		t.Fatalf("status %q", got)
	}
	rev := b.Group(src.RelPath).Passages[0].SourceRevision
	b.Observe(src, time.Unix(10, 0))
	b.Observe(src, time.Unix(10, 0))
	if b.Group(src.RelPath).Passages[0].SourceRevision != rev {
		t.Fatal("observe bumped a revision without an edit")
	}
	if strings.Contains(tgt.CommittedMarkdown(), "Hello") && strings.Contains(tgt.CommittedMarkdown(), "World") {
		t.Fatalf("target copied the source:\n%s", tgt.CommittedMarkdown())
	}
}

func TestSourceEditMarksOutdatedAndPause(t *testing.T) {
	b, src, _ := linked(t)
	id := src.Passages()[0].ID
	src.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "J", richtext.Attributes{})
	})
	now := time.Unix(100, 0)
	b.Observe(src, now)
	if got := b.Status(src.RelPath, id, model.LangItalian); got != "outdated" {
		t.Fatalf("status %q", got)
	}
	if len(b.Candidates(src, now)) != 0 {
		t.Fatal("candidate before the pause")
	}
	got := b.Candidates(src, now.Add(DefaultPause))
	if len(got) != 1 || got[0].PassageID != id || got[0].Target != model.LangItalian {
		t.Fatalf("candidates %+v", got)
	}
	b.SetVisible(model.LangRussian)
	if len(b.Candidates(src, now.Add(DefaultPause))) != 0 {
		t.Fatal("hidden Italian was eligible for an automatic update")
	}
	on, err := b.RequestOnDemand(src, id, model.LangItalian, now)
	if err != nil || !on.OnDemand || on.Target != model.LangItalian {
		t.Fatalf("on demand %+v %v", on, err)
	}
}

func TestDeliverRejectsStaleProtectedAndCancel(t *testing.T) {
	b, src, tgt := linked(t)
	id := src.Passages()[0].ID
	src.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "J", richtext.Attributes{})
	})
	now := time.Unix(1, 0)
	b.Observe(src, now)
	reqs := b.Candidates(src, now.Add(DefaultPause))
	if len(reqs) != 1 {
		t.Fatal(reqs)
	}
	first := reqs[0]
	b.Accept(first)
	src.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "Q", richtext.Attributes{})
	})
	b.Observe(src, now.Add(time.Second))
	if err := b.Deliver(tgt, Result{Request: first, Text: "FAKE"}); err != ErrStale {
		t.Fatalf("mid-edit deliver %v", err)
	}
	if strings.Contains(tgt.Rich.Text().String(), "FAKE") {
		t.Fatal("stale result was written")
	}
	second := b.Candidates(src, now.Add(time.Second+DefaultPause))
	if len(second) != 1 {
		t.Fatal(second)
	}
	b.Accept(second[0])
	if err := b.Deliver(tgt, Result{Request: first, Text: "OLD"}); err != ErrStale {
		t.Fatalf("out of order %v", err)
	}
	b.SetFocus("other.md")
	if err := b.Deliver(tgt, Result{Request: second[0], Text: "NO"}); err != ErrStale {
		t.Fatalf("other document %v", err)
	}
	b.SetFocus(src.RelPath)
	again := b.Candidates(src, now.Add(2*time.Second+DefaultPause))
	if len(again) != 1 {
		t.Fatal(again)
	}
	b.Accept(again[0])
	b.Cancel(again[0].ID)
	if err := b.Deliver(tgt, Result{Request: again[0], Text: "NO"}); err != ErrCancelled {
		t.Fatalf("cancel %v", err)
	}

	third := b.Candidates(src, now.Add(2*time.Second+DefaultPause))
	if len(third) != 1 {
		t.Fatal(third)
	}
	b.Accept(third[0])
	tgt.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "manuale", richtext.Attributes{})
	})
	b.Observe(tgt, now)
	if got := b.Status(src.RelPath, id, model.LangItalian); got != "pending, protected, outdated" {
		t.Fatalf("status %q", got)
	}
	before := tgt.Rich.Text().String()
	if err := b.Deliver(tgt, Result{Request: third[0], Text: "FAKE"}); err != ErrProtected {
		t.Fatalf("protected deliver %v", err)
	}
	if tgt.Rich.Text().String() != before {
		t.Fatal("protected text changed")
	}
}

func TestExplicitReplaceUndoRestoresProtection(t *testing.T) {
	b, src, tgt := linked(t)
	id := src.Passages()[0].ID
	tgt.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "manuale", richtext.Attributes{})
	})
	b.Observe(tgt, time.Unix(1, 0))
	if err := b.Replace(tgt, id, "ciao"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tgt.Rich.Text().String(), "ciao") {
		t.Fatalf("text %q", tgt.Rich.Text().String())
	}
	if b.Status(src.RelPath, id, model.LangItalian) != "current" {
		t.Fatalf("after replace %q", b.Status(src.RelPath, id, model.LangItalian))
	}
	if err := b.UndoReplace(tgt); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(tgt.Rich.Text().String(), "ciao") || !strings.Contains(tgt.Rich.Text().String(), "manuale") {
		t.Fatalf("undo text %q", tgt.Rich.Text().String())
	}
	if got := b.Status(src.RelPath, id, model.LangItalian); got != "protected, outdated" {
		t.Fatalf("undo status %q", got)
	}
	tgt.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "x", richtext.Attributes{})
	})
	if err := b.UndoReplace(tgt); err != ErrStale {
		t.Fatalf("undo after a later edit %v", err)
	}
}

func TestLeaveAndSaveTriggers(t *testing.T) {
	b, src, _ := linked(t)
	id := src.Passages()[0].ID
	b.SetTrigger(TriggerLeave)
	src.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "J", richtext.Attributes{})
	})
	b.Observe(src, time.Unix(1, 0))
	if len(b.Candidates(src, time.Unix(1, 0).Add(time.Hour))) != 0 {
		t.Fatal("leave trigger fired before the caret left")
	}
	b.MarkLeft(id)
	got := b.Candidates(src, time.Unix(1, 0))
	if len(got) != 1 {
		t.Fatal(got)
	}
	b2, src2, _ := linked(t)
	id2 := src2.Passages()[0].ID
	b2.SetTrigger(TriggerSave)
	src2.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "J", richtext.Attributes{})
	})
	b2.Observe(src2, time.Unix(1, 0))
	if len(b2.Candidates(src2, time.Unix(1, 0).Add(time.Hour))) != 0 {
		t.Fatal("save trigger fired before save")
	}
	b2.NoteSaved()
	got = b2.Candidates(src2, time.Unix(1, 0))
	if len(got) != 1 || got[0].PassageID != id2 {
		t.Fatal(got)
	}
}

func TestTableDocumentIsNotLinked(t *testing.T) {
	for _, body := range []string{
		"| a | b |\n| --- | --- |\n| 1 | 2 |\n",
		"Before\n\n<table><tr><td>a</td></tr></table>\n",
	} {
		src := model.OpenMarkdown("t.md", body, "", model.LangEnglish, nil)
		if _, _, _, err := PlanTarget(src, model.LangItalian); err == nil {
			t.Fatalf("linked unsupported source %q", body)
		}
	}
	src := model.OpenMarkdown("notes/Hello.md", "Hello\n", "", model.LangEnglish, nil)
	if err := src.SwitchMode(model.ModeSource); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := PlanTarget(src, model.LangItalian); err == nil {
		t.Fatal("linked a document that is showing Markdown source")
	}
}

func TestPendingHiddenAndReplaceDuringRequest(t *testing.T) {
	b, src, tgt := linked(t)
	id := src.Passages()[0].ID
	src.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "J", richtext.Attributes{})
	})
	now := time.Unix(5, 0)
	b.Observe(src, now)
	reqs := b.Candidates(src, now.Add(DefaultPause))
	if len(reqs) != 1 {
		t.Fatal(reqs)
	}
	b.Accept(reqs[0])
	if got := b.Status(src.RelPath, id, model.LangItalian); got != "pending, outdated" {
		t.Fatalf("pending status %q", got)
	}
	b.SetVisible(model.LangRussian)
	if err := b.Deliver(tgt, Result{Request: reqs[0], Text: "FAKE"}); err != ErrStale {
		t.Fatalf("hidden auto result %v", err)
	}
	if strings.Contains(tgt.Rich.Text().String(), "FAKE") {
		t.Fatal("hidden target accepted an automatic result")
	}
	if strings.Contains(b.Status(src.RelPath, id, model.LangItalian), "pending") {
		t.Fatal("a rejected request stayed pending")
	}
	on, err := b.RequestOnDemand(src, id, model.LangItalian, now)
	if err != nil {
		t.Fatal(err)
	}
	b.Accept(on)
	if err := b.Deliver(src, Result{Request: on, Text: "FAKE"}); err != ErrStale {
		t.Fatalf("deliver into the source %v", err)
	}
	if strings.Contains(src.Rich.Text().String(), "FAKE") || strings.Contains(tgt.Rich.Text().String(), "FAKE") {
		t.Fatal("a result for the wrong document was written")
	}
	on, err = b.RequestOnDemand(src, id, model.LangItalian, now)
	if err != nil {
		t.Fatal(err)
	}
	b.Accept(on)
	if err := b.Replace(tgt, id, "ciao"); err != nil {
		t.Fatal(err)
	}
	if err := b.Deliver(tgt, Result{Request: on, Text: "FAKE"}); err != ErrStale {
		t.Fatalf("deliver after replace %v", err)
	}
	if strings.Contains(tgt.Rich.Text().String(), "FAKE") || !strings.Contains(tgt.Rich.Text().String(), "ciao") {
		t.Fatalf("text %q", tgt.Rich.Text().String())
	}

	b.SetVisible(model.LangItalian)
	src.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "K", richtext.Attributes{})
	})
	b.Observe(src, now.Add(time.Minute))
	again := b.Candidates(src, now.Add(time.Minute+DefaultPause))
	if len(again) != 1 {
		t.Fatal(again)
	}
	b.Accept(again[0])
	b.Cancel(again[0].ID)
	src.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, 1, "Z", richtext.Attributes{})
	})
	b.Observe(src, now.Add(2*time.Minute))
	if err := b.Deliver(tgt, Result{Request: again[0], Text: "FAKE"}); err != ErrStale {
		t.Fatalf("cancelled and stale %v", err)
	}
	if strings.Contains(tgt.Rich.Text().String(), "FAKE") {
		t.Fatal("stale cancelled result was written")
	}
}

func TestSyncChoice(t *testing.T) {
	moveT, moveS, next := SyncChoice(true, "b", "a", "a")
	if moveT != "b" || moveS != "" || next != "b" {
		t.Fatalf("source move %s %s %s", moveT, moveS, next)
	}
	moveT, moveS, next = SyncChoice(true, "b", "a", "b")
	if moveT != "" || moveS != "a" || next != "a" {
		t.Fatalf("target move %s %s %s", moveT, moveS, next)
	}
	moveT, _, next = SyncChoice(false, "b", "a", "a")
	if moveT != "" || next != "b" {
		t.Fatalf("toggle off %s %s", moveT, next)
	}
}
