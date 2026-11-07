// Package model is the open document: rich text or Markdown source, passage
// IDs, and the rule that a lossy import is refused instead of saved over
// the original.
package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui/richtext"
)

const Schema = 1

type Language string

const (
	LangEnglish  Language = "en"
	LangItalian  Language = "it"
	LangRussian  Language = "ru"
	LangJapanese Language = "ja"
)

func ParseLanguage(s string) (Language, error) {
	switch Language(s) {
	case LangEnglish, LangItalian, LangRussian, LangJapanese:
		return Language(s), nil
	default:
		return "", fmt.Errorf("language %q is not one of en, it, ru, ja", s)
	}
}

func (l Language) Label() string {
	switch l {
	case LangItalian:
		return "Italian"
	case LangRussian:
		return "Russian"
	case LangJapanese:
		return "Japanese"
	default:
		return "English"
	}
}

type Mode string

const (
	ModeRich   Mode = "rich"
	ModeSource Mode = "source"
)

type Fidelity string

const (
	// FidelityRich means the Markdown uses only constructs the rich editor
	// can export again without dropping a table.
	FidelityRich Fidelity = "rich"
	// FidelitySource means the file stays in Markdown source. Rich text is
	// refused so a table is not flattened and written back.
	FidelitySource Fidelity = "source"
)

var (
	ErrLossy      = errors.New("rich text would drop unsupported Markdown")
	ErrNoDocument = errors.New("no document")
)

// Passage is a stable id for one paragraph, stored beside the Markdown.
// Markdown itself does not contain the id. A full source replacement keeps
// ids by paragraph order and assigns new ids to extra paragraphs.
type Passage struct {
	ID string `json:"id"`
}

// Session is one open document. Rich undo lives on Rich. Switching to
// source does not call SetText, so that undo stack stays until a real
// source edit is applied back with Document.Edit.
type Session struct {
	ID          string
	RelPath     string
	Language    Language
	Mode        Mode
	Fidelity    Fidelity
	Notice      string
	Rich        *richtext.Document
	Source      string
	savedDigest string
	savedLang   Language
	baseline    string // source exported when rich mode was left
	anchor      int
	caret       int
}

func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

func Digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// OpenMarkdown builds a session from file bytes and sidecar passages.
// Unsupported Markdown opens in source mode and is not parsed into the
// rich editor.
func OpenMarkdown(rel string, markdown string, id string, lang Language, passages []Passage) *Session {
	if id == "" {
		id = NewID()
	}
	if lang == "" {
		lang = LangEnglish
	}
	s := &Session{
		ID:          id,
		RelPath:     rel,
		Language:    lang,
		savedDigest: Digest(markdown),
		savedLang:   lang,
	}
	if reasons := Unsupported(markdown); len(reasons) > 0 {
		s.Fidelity = FidelitySource
		s.Mode = ModeSource
		s.Source = markdown
		s.Notice = "Rich text is off: this file has a " + strings.Join(reasons, " and a ") + ". Editing stays in Markdown so the original is not rewritten as paragraphs."
		return s
	}
	s.Fidelity = FidelityRich
	s.Mode = ModeRich
	s.Rich = richFromMarkdown(markdown, passages)
	s.anchor, s.caret = s.Rich.Selection()
	return s
}

func richFromMarkdown(markdown string, passages []Passage) *richtext.Document {
	text := stampPassages(richtext.ParseMarkdown(markdown), passages)
	doc := richtext.NewDocument(text)
	doc.IdentifyParagraphs(NewID)
	return doc
}

func stampPassages(text richtext.Text, passages []Passage) richtext.Text {
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

// Passages lists paragraph ids in order. Source-only documents have none
// until a later milestone maps them without a lossy parse.
func (s *Session) Passages() []Passage {
	if s == nil || s.Rich == nil || s.Fidelity != FidelityRich {
		return nil
	}
	var out []Passage
	for _, p := range s.Rich.Text().Paragraphs() {
		if p.Style.ID == "" {
			continue
		}
		out = append(out, Passage{ID: p.Style.ID})
	}
	return out
}

// CommittedMarkdown is the text that may be saved. Rich-text composition
// is not part of Document.Text, so this excludes pre-edit.
func (s *Session) CommittedMarkdown() string {
	if s.Fidelity == FidelitySource || s.Mode == ModeSource {
		return s.Source
	}
	if s.Rich == nil {
		return ""
	}
	return s.Rich.Text().Markdown()
}

func (s *Session) Dirty() bool {
	if s.Language != s.savedLang {
		return true
	}
	return Digest(s.CommittedMarkdown()) != s.savedDigest
}

func (s *Session) MarkSaved() {
	s.savedDigest = Digest(s.CommittedMarkdown())
	s.savedLang = s.Language
}

func (s *Session) SavedDigest() string { return s.savedDigest }

// ForceDirty marks a recovered session unsaved relative to the bytes last
// written to the Markdown file.
func (s *Session) ForceDirty(savedDigest string) {
	s.savedDigest = savedDigest
}

func (s *Session) Selection() (anchor, caret int) {
	if s.Mode == ModeRich && s.Rich != nil {
		return s.Rich.Selection()
	}
	return s.anchor, s.caret
}

func (s *Session) SetSelection(anchor, caret int) {
	s.anchor, s.caret = anchor, caret
	if s.Mode == ModeRich && s.Rich != nil {
		s.Rich.SetSelection(anchor, caret)
	}
}

// SwitchMode moves between rich text and Markdown source.
// Leaving rich exports Markdown and does not call SetText.
// Returning without a source edit leaves the rich document, including its
// undo stack, alone. A changed source is applied with Document.Edit, or
// refused when the source now contains a table.
func (s *Session) SwitchMode(to Mode) error {
	if s.Fidelity == FidelitySource && to == ModeRich {
		return fmt.Errorf("%w: %s", ErrLossy, s.Notice)
	}
	if to == s.Mode {
		return nil
	}
	switch to {
	case ModeSource:
		s.Source = s.Rich.Text().Markdown()
		s.baseline = s.Source
		s.anchor, s.caret = s.Rich.Selection()
		s.Mode = ModeSource
		s.Notice = "Markdown source. Rich-text undo is kept if you switch back without editing this source."
		return nil
	case ModeRich:
		if s.Source == s.baseline && s.Rich != nil {
			s.Mode = ModeRich
			s.Notice = ""
			s.Rich.SetSelection(s.anchor, s.caret)
			return nil
		}
		if reasons := Unsupported(s.Source); len(reasons) > 0 {
			s.Notice = "Staying in Markdown source: " + strings.Join(reasons, ", ") + " would be dropped in rich text."
			return fmt.Errorf("%w: %s", ErrLossy, s.Notice)
		}
		parsed := stampPassages(richtext.ParseMarkdown(s.Source), s.Passages())
		saved := passageQueue(s.Passages())
		s.Rich.IdentifyParagraphs(func() string {
			if len(saved) > 0 {
				id := saved[0]
				saved = saved[1:]
				return id
			}
			return NewID()
		})
		anchor, caret := s.anchor, s.caret
		s.Rich.Edit(func(e *richtext.Edit) {
			n := e.Text().Len()
			e.Replace(0, n, parsed)
			limit := e.Text().Len()
			if anchor > limit {
				anchor = limit
			}
			if caret > limit {
				caret = limit
			}
			e.Select(anchor, caret)
		})
		s.Rich.IdentifyParagraphs(NewID)
		s.Mode = ModeRich
		s.baseline = ""
		s.Notice = "Source edits were applied as one undo step."
		return nil
	default:
		return fmt.Errorf("unknown mode %q", to)
	}
}

func passageQueue(ps []Passage) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		if p.ID != "" {
			out = append(out, p.ID)
		}
	}
	return out
}

// NoteSource records the textarea's committed string. Call it only when
// the field is not composing, so pre-edit is not stored as the document.
func (s *Session) NoteSource(committed string, anchor, caret int) {
	s.Source = committed
	s.anchor, s.caret = anchor, caret
}
