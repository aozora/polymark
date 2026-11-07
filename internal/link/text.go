package link

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/egoist/mygo/ui/richtext"

	"github.com/aozora/polymark/internal/model"
)

type span struct {
	ID         string
	Start      int
	ContentEnd int
	Plain      string
}

func spansOf(text richtext.Text) []span {
	type item struct {
		start int
		p     richtext.Paragraph
	}
	var items []item
	for start, p := range text.Paragraphs() {
		items = append(items, item{start, p})
	}
	n := text.Len()
	runes := []rune(text.String())
	out := make([]span, 0, len(items))
	for i, it := range items {
		end := n
		if i+1 < len(items) {
			end = items[i+1].start - 1
			if end < it.start {
				end = it.start
			}
		}
		if end > len(runes) {
			end = len(runes)
		}
		start := it.start
		if start > len(runes) {
			start = len(runes)
		}
		if end < start {
			end = start
		}
		out = append(out, span{
			ID:         it.p.Style.ID,
			Start:      start,
			ContentEnd: end,
			Plain:      string(runes[start:end]),
		})
	}
	return out
}

func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// PassageStart returns the first rune of the named passage.
func PassageStart(s *model.Session, id string) (int, bool) {
	text, ok := richText(s)
	if !ok {
		return 0, false
	}
	for _, sp := range spansOf(text) {
		if sp.ID == id {
			return sp.Start, true
		}
	}
	return 0, false
}

// PassageAt returns the passage id and start rune containing caret.
func PassageAt(s *model.Session, caret int) (id string, start int, ok bool) {
	text, ok := richText(s)
	if !ok {
		return "", 0, false
	}
	var found span
	any := false
	for _, sp := range spansOf(text) {
		if sp.ID == "" {
			continue
		}
		if caret >= sp.Start {
			found = sp
			any = true
		}
	}
	if !any {
		return "", 0, false
	}
	return found.ID, found.Start, true
}

func richText(s *model.Session) (richtext.Text, bool) {
	if s == nil || s.Fidelity != model.FidelityRich || s.Rich == nil {
		return richtext.Text{}, false
	}
	if s.Mode == model.ModeSource {
		if len(model.Unsupported(s.Source)) > 0 {
			return richtext.Text{}, false
		}
		return stamp(richtext.ParseMarkdown(s.Source), s.Passages()), true
	}
	return s.Rich.Text(), true
}

func stamp(text richtext.Text, passages []model.Passage) richtext.Text {
	if len(passages) == 0 {
		return text
	}
	var paras []richtext.Paragraph
	i := 0
	for _, p := range text.Paragraphs() {
		if i < len(passages) && passages[i].ID != "" && p.Style.ID == "" {
			p.Style.ID = passages[i].ID
		}
		paras = append(paras, p)
		i++
	}
	if len(paras) == 0 {
		return text
	}
	return richtext.TextOf(paras...)
}
