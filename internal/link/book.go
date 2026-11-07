// Package link records which documents translate which, by passage id.
// It does not call a translation provider. A nil provider means the app
// is not configured, and tests supply results directly.
package link

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/egoist/mygo/ui/richtext"

	"github.com/aozora/polymark/internal/model"
)

const Schema = 1

const DefaultPause = 400 * time.Millisecond

// emptyPassage is a non-breaking space. A normal space is trimmed when
// Markdown is parsed, and the target paragraphs would collapse into one.
const emptyPassage = "\u00a0"

type Trigger string

const (
	TriggerPause Trigger = "pause"
	TriggerLeave Trigger = "leave"
	TriggerSave  Trigger = "save"
)

var (
	ErrStale     = errors.New("stale translation result")
	ErrProtected = errors.New("passage is protected")
	ErrCancelled = errors.New("translation request was cancelled")
	ErrNoGroup   = errors.New("document has no linked translations")
	ErrLossy     = model.ErrLossy
)

type TargetLink struct {
	Revision      uint64 `json:"revision"`
	AppliedSource uint64 `json:"appliedSource"`
	Protected     bool   `json:"protected,omitempty"`
	TextHash      string `json:"textHash,omitempty"`
}

type PassageLink struct {
	ID             string                `json:"id"`
	SourceRevision uint64                `json:"sourceRevision"`
	SourceHash     string                `json:"sourceHash,omitempty"`
	Targets        map[string]TargetLink `json:"targets,omitempty"`
}

type Group struct {
	ID       string            `json:"id"`
	Source   string            `json:"source"`
	Passages []PassageLink     `json:"passages"`
	Targets  map[string]string `json:"targets"` // language -> relative path
}

type File struct {
	Schema         int     `json:"schema"`
	ScrollTogether *bool   `json:"scrollTogether,omitempty"`
	Trigger        string  `json:"trigger,omitempty"`
	Groups         []Group `json:"groups,omitempty"`
}

// Request is a provider-independent translation ask. The text to send is
// the source passage. Nothing in this package contacts the network.
type Request struct {
	ID             string
	GroupID        string
	DocRel         string
	PassageID      string
	Target         model.Language
	SourceRevision uint64
	TargetRevision uint64
	OnDemand       bool
	Text           string
}

type Result struct {
	Request Request
	Text    string
}

type undoItem struct {
	rel     string
	passage string
	lang    model.Language
	version uint64
	before  TargetLink
	plain   string
}

type ticket struct {
	req       Request
	cancelled bool
	done      bool
}

type Book struct {
	file      File
	visible   model.Language
	focus     string
	pending   map[string]*ticket
	undo      []undoItem
	dirty     bool
	pause     time.Duration
	edited    map[string]time.Time // passage id -> source change time
	leave     map[string]bool      // source text changed, waiting to leave the passage
	leaveDue  map[string]bool
	saveDue   map[string]bool // source text changed, waiting for a save
	saveReady map[string]bool
	held      map[string]string // "lang/id" -> plain text just written by Replace
}

func New() *Book {
	return &Book{
		file:      File{Schema: Schema},
		pause:     DefaultPause,
		pending:   map[string]*ticket{},
		edited:    map[string]time.Time{},
		leave:     map[string]bool{},
		leaveDue:  map[string]bool{},
		saveDue:   map[string]bool{},
		saveReady: map[string]bool{},
		held:      map[string]string{},
	}
}

func (b *Book) Load(f File) {
	if f.Schema == 0 {
		f.Schema = Schema
	}
	b.file = f
	b.dirty = false
}

func (b *Book) File() File {
	f := b.file
	f.Schema = Schema
	if b.Trigger() != TriggerPause {
		f.Trigger = string(b.Trigger())
	}
	return f
}

func (b *Book) Dirty() bool { return b.dirty }

func (b *Book) MarkSaved() { b.dirty = false }

func (b *Book) Trigger() Trigger {
	switch Trigger(b.file.Trigger) {
	case TriggerLeave, TriggerSave:
		return Trigger(b.file.Trigger)
	default:
		return TriggerPause
	}
}

func (b *Book) SetTrigger(t Trigger) {
	if b.Trigger() == t {
		return
	}
	b.file.Trigger = string(t)
	b.dirty = true
}

func (b *Book) ScrollTogether() bool {
	if b.file.ScrollTogether == nil {
		return true
	}
	return *b.file.ScrollTogether
}

func (b *Book) SetScrollTogether(on bool) {
	b.file.ScrollTogether = &on
	b.dirty = true
}

func (b *Book) SetVisible(lang model.Language) {
	b.visible = lang
	b.dropInflight(func(req Request) bool {
		return !req.OnDemand && req.Target != lang
	})
}

func (b *Book) Visible() model.Language { return b.visible }

func (b *Book) SetFocus(sourceRel string) {
	b.focus = sourceRel
	b.dropInflight(func(req Request) bool {
		return req.DocRel != sourceRel
	})
}

func (b *Book) dropInflight(drop func(Request) bool) {
	for _, t := range b.pending {
		if t.done || t.cancelled {
			continue
		}
		if drop(t.req) {
			t.cancelled = true
		}
	}
}

func (b *Book) Focus() string { return b.focus }

func (b *Book) Group(sourceRel string) *Group {
	for i := range b.file.Groups {
		if b.file.Groups[i].Source == sourceRel {
			return &b.file.Groups[i]
		}
	}
	return nil
}

func (b *Book) GroupByPath(rel string) *Group {
	for i := range b.file.Groups {
		g := &b.file.Groups[i]
		if g.Source == rel {
			return g
		}
		for _, path := range g.Targets {
			if path == rel {
				return g
			}
		}
	}
	return nil
}

func (b *Book) TargetPath(sourceRel string, lang model.Language) string {
	g := b.Group(sourceRel)
	if g == nil {
		return ""
	}
	return g.Targets[string(lang)]
}

func (b *Book) LanguageOf(rel string) (model.Language, bool) {
	g := b.GroupByPath(rel)
	if g == nil {
		return "", false
	}
	if g.Source == rel {
		return "", false
	}
	for lang, path := range g.Targets {
		if path == rel {
			return model.Language(lang), true
		}
	}
	return "", false
}

// PlanTarget describes a new empty target file. It does not write it.
func PlanTarget(source *model.Session, lang model.Language) (rel, markdown string, passages []model.Passage, err error) {
	if source == nil {
		return "", "", nil, model.ErrNoDocument
	}
	if source.Fidelity != model.FidelityRich || source.Mode != model.ModeRich || source.Rich == nil {
		return "", "", nil, fmt.Errorf("%w: link a document from rich text; Markdown source, including tables, stays unlinked", model.ErrLossy)
	}
	if source.Language == lang {
		return "", "", nil, fmt.Errorf("target language %s is the document language", lang.Label())
	}
	passages = source.Passages()
	if len(passages) == 0 {
		return "", "", nil, errors.New("source has no passage ids")
	}
	rel = targetRel(source.RelPath, lang)
	var paras []richtext.Paragraph
	for _, p := range passages {
		paras = append(paras, richtext.Paragraph{
			Style: richtext.ParagraphStyle{ID: p.ID},
			Runs:  []richtext.Run{{Text: emptyPassage}},
		})
	}
	return rel, richtext.TextOf(paras...).Markdown(), passages, nil
}

func targetRel(source string, lang model.Language) string {
	base := source
	if len(base) > 3 && base[len(base)-3:] == ".md" {
		base = base[:len(base)-3]
	}
	return base + "." + string(lang) + ".md"
}

// AdoptTarget records a target that was just created or already on disk.
func (b *Book) AdoptTarget(source *model.Session, target *model.Session) error {
	if source == nil || target == nil {
		return model.ErrNoDocument
	}
	if source.Fidelity != model.FidelityRich || target.Fidelity != model.FidelityRich {
		return model.ErrLossy
	}
	g := b.Group(source.RelPath)
	if g == nil {
		g = &Group{
			ID:      newID(),
			Source:  source.RelPath,
			Targets: map[string]string{},
		}
		b.file.Groups = append(b.file.Groups, *g)
		g = &b.file.Groups[len(b.file.Groups)-1]
	}
	if g.Targets == nil {
		g.Targets = map[string]string{}
	}
	g.Targets[string(target.Language)] = target.RelPath
	b.rebuildPassages(g, source)
	b.observeSession(source, time.Time{})
	b.observeSession(target, time.Time{})
	b.dirty = true
	return nil
}

func (b *Book) rebuildPassages(g *Group, source *model.Session) {
	seen := map[string]bool{}
	for _, sp := range spansOf(source.Rich.Text()) {
		if sp.ID == "" {
			continue
		}
		seen[sp.ID] = true
		p := b.passage(g, sp.ID)
		if p.SourceHash == "" {
			p.SourceHash = hashText(sp.Plain)
			if p.SourceRevision == 0 {
				p.SourceRevision = 1
			}
		}
		b.putPassage(g, p)
	}
}

func (b *Book) Observe(s *model.Session, now time.Time) {
	if s == nil {
		return
	}
	b.observeSession(s, now)
}

func (b *Book) observeSession(s *model.Session, now time.Time) {
	g := b.GroupByPath(s.RelPath)
	if g == nil || s.Fidelity != model.FidelityRich {
		return
	}
	text, ok := richText(s)
	if !ok {
		return
	}
	if g.Source == s.RelPath {
		for _, sp := range spansOf(text) {
			if sp.ID == "" {
				continue
			}
			p := b.passage(g, sp.ID)
			h := hashText(sp.Plain)
			if p.SourceHash == "" {
				p.SourceHash = h
				if p.SourceRevision == 0 {
					p.SourceRevision = 1
				}
				b.putPassage(g, p)
				continue
			}
			if h == p.SourceHash {
				continue
			}
			p.SourceHash = h
			p.SourceRevision++
			b.putPassage(g, p)
			if !now.IsZero() {
				b.edited[sp.ID] = now
			}
			b.leave[sp.ID] = true
			b.saveDue[sp.ID] = true
			b.dirty = true
		}
		return
	}
	lang, ok := b.LanguageOf(s.RelPath)
	if !ok {
		return
	}
	for _, sp := range spansOf(text) {
		if sp.ID == "" {
			continue
		}
		p := b.passage(g, sp.ID)
		t := p.Targets[string(lang)]
		h := hashText(sp.Plain)
		key := string(lang) + "/" + sp.ID
		if held, ok := b.held[key]; ok && held == sp.Plain {
			delete(b.held, key)
			t.TextHash = h
			if p.Targets == nil {
				p.Targets = map[string]TargetLink{}
			}
			p.Targets[string(lang)] = t
			b.putPassage(g, p)
			continue
		}
		if t.TextHash == "" {
			t.TextHash = h
			if p.Targets == nil {
				p.Targets = map[string]TargetLink{}
			}
			p.Targets[string(lang)] = t
			b.putPassage(g, p)
			continue
		}
		if h == t.TextHash {
			continue
		}
		t.TextHash = h
		t.Protected = true
		t.Revision++
		if p.Targets == nil {
			p.Targets = map[string]TargetLink{}
		}
		p.Targets[string(lang)] = t
		b.putPassage(g, p)
		b.dirty = true
	}
}

// MarkLeft arms passages whose source text changed once the caret leaves them.
func (b *Book) MarkLeft(passageID string) {
	if passageID == "" || !b.leave[passageID] {
		return
	}
	b.leaveDue[passageID] = true
}

func (b *Book) NoteSaved() {
	for id, on := range b.saveDue {
		if on {
			b.saveReady[id] = true
		}
	}
}

func (b *Book) Status(sourceRel, passageID string, lang model.Language) string {
	g := b.Group(sourceRel)
	if g == nil {
		return ""
	}
	p := b.passage(g, passageID)
	if p.ID == "" {
		return ""
	}
	t := p.Targets[string(lang)]
	state := labels(p.SourceRevision, t)
	if b.inflight(g.ID, passageID, lang) {
		if state == "current" {
			return "pending"
		}
		return "pending, " + state
	}
	return state
}

func labels(sourceRev uint64, t TargetLink) string {
	var parts []string
	if t.Protected {
		parts = append(parts, "protected")
	}
	if t.AppliedSource < sourceRev {
		parts = append(parts, "outdated")
	}
	if len(parts) == 0 {
		return "current"
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += ", " + p
	}
	return out
}

func (b *Book) Candidates(source *model.Session, now time.Time) []Request {
	if b.visible == "" || b.focus == "" || source == nil {
		return nil
	}
	g := b.Group(b.focus)
	if g == nil || g.Source != source.RelPath {
		return nil
	}
	plain := map[string]string{}
	if text, ok := richText(source); ok {
		for _, sp := range spansOf(text) {
			plain[sp.ID] = sp.Plain
		}
	}
	var out []Request
	for _, p := range g.Passages {
		req, ok := b.candidate(g, p, b.visible, false, now)
		if ok {
			req.Text = plain[p.ID]
			out = append(out, req)
		}
	}
	return out
}

func (b *Book) RequestOnDemand(source *model.Session, passageID string, lang model.Language, now time.Time) (Request, error) {
	g := b.Group(b.focus)
	if g == nil || source == nil {
		return Request{}, ErrNoGroup
	}
	p := b.passage(g, passageID)
	if p.ID == "" {
		return Request{}, fmt.Errorf("unknown passage %s", passageID)
	}
	req, ok := b.candidate(g, p, lang, true, now)
	if !ok {
		return Request{}, ErrStale
	}
	req.OnDemand = true
	if text, ok := richText(source); ok {
		for _, sp := range spansOf(text) {
			if sp.ID == passageID {
				req.Text = sp.Plain
			}
		}
	}
	return req, nil
}

func (b *Book) candidate(g *Group, p PassageLink, lang model.Language, demand bool, now time.Time) (Request, bool) {
	if g.Targets[string(lang)] == "" {
		return Request{}, false
	}
	t := p.Targets[string(lang)]
	if t.Protected || t.AppliedSource >= p.SourceRevision {
		return Request{}, false
	}
	if !demand && lang != b.visible {
		return Request{}, false
	}
	if !demand && !b.triggerReady(p.ID, now) {
		return Request{}, false
	}
	if b.inflight(g.ID, p.ID, lang) {
		return Request{}, false
	}
	return Request{
		ID:             newID(),
		GroupID:        g.ID,
		DocRel:         g.Source,
		PassageID:      p.ID,
		Target:         lang,
		SourceRevision: p.SourceRevision,
		TargetRevision: t.Revision,
	}, true
}

func (b *Book) triggerReady(id string, now time.Time) bool {
	switch b.Trigger() {
	case TriggerLeave:
		return b.leaveDue[id]
	case TriggerSave:
		return b.saveReady[id]
	default:
		at, ok := b.edited[id]
		if !ok || now.IsZero() {
			return false
		}
		return !now.Before(at.Add(b.pause))
	}
}

func (b *Book) inflight(group, passage string, lang model.Language) bool {
	for _, t := range b.pending {
		if !t.done && !t.cancelled && t.req.GroupID == group && t.req.PassageID == passage && t.req.Target == lang {
			return true
		}
	}
	return false
}

func (b *Book) Accept(req Request) {
	b.pending[req.ID] = &ticket{req: req}
}

func (b *Book) Cancel(id string) {
	if t, ok := b.pending[id]; ok {
		t.cancelled = true
	}
}

func (b *Book) Deliver(target *model.Session, res Result) error {
	tkt, ok := b.pending[res.Request.ID]
	if !ok {
		return ErrStale
	}
	if tkt.done {
		return ErrStale
	}
	req := tkt.req
	finish := func(err error) error {
		tkt.done = true
		return err
	}
	// Mismatch is rejected even when the ticket was also cancelled.
	if res.Request.GroupID != "" && res.Request.GroupID != req.GroupID {
		return finish(ErrStale)
	}
	if b.focus != "" && req.DocRel != b.focus {
		return finish(ErrStale)
	}
	if !req.OnDemand && b.visible != "" && req.Target != b.visible {
		return finish(ErrStale)
	}
	g := b.groupByID(req.GroupID)
	if g == nil || g.Source != req.DocRel {
		return finish(ErrStale)
	}
	if target == nil || g.Targets[string(req.Target)] != target.RelPath {
		return finish(ErrStale)
	}
	p := b.passage(g, req.PassageID)
	if p.ID == "" || p.SourceRevision != req.SourceRevision {
		return finish(ErrStale)
	}
	cur := p.Targets[string(req.Target)]
	if cur.Protected {
		return finish(ErrProtected)
	}
	if cur.Revision != req.TargetRevision {
		return finish(ErrStale)
	}
	if tkt.cancelled {
		return finish(ErrCancelled)
	}
	if err := b.apply(target, req.PassageID, req.Target, res.Text, p.SourceRevision, false); err != nil {
		tkt.done = true
		return err
	}
	tkt.done = true
	return nil
}

// Replace is the explicit user replacement of one linked passage.
func (b *Book) Replace(target *model.Session, passageID, text string) error {
	g := b.GroupByPath(target.RelPath)
	if g == nil {
		return ErrNoGroup
	}
	lang, ok := b.LanguageOf(target.RelPath)
	if !ok {
		return ErrNoGroup
	}
	p := b.passage(g, passageID)
	if p.ID == "" {
		return ErrStale
	}
	return b.apply(target, passageID, lang, text, p.SourceRevision, true)
}

func (b *Book) apply(target *model.Session, passageID string, lang model.Language, text string, sourceRev uint64, clearProtect bool) error {
	if target == nil || target.Rich == nil {
		return model.ErrNoDocument
	}
	g := b.GroupByPath(target.RelPath)
	if g == nil {
		return ErrNoGroup
	}
	sp, ok := spanByID(target.Rich.Text(), passageID)
	if !ok {
		return ErrStale
	}
	if text == "" {
		text = emptyPassage
	}
	before := b.passage(g, passageID).Targets[string(lang)]
	version := uint64(0)
	target.Rich.Edit(func(e *richtext.Edit) {
		e.ReplaceString(sp.Start, sp.ContentEnd, text, richtext.Attributes{})
	})
	version = target.Rich.Version()
	b.held[string(lang)+"/"+passageID] = text
	next := before
	next.TextHash = hashText(text)
	next.AppliedSource = sourceRev
	next.Revision++
	if clearProtect {
		next.Protected = false
	}
	p := b.passage(g, passageID)
	if p.Targets == nil {
		p.Targets = map[string]TargetLink{}
	}
	p.Targets[string(lang)] = next
	b.putPassage(g, p)
	b.undo = append(b.undo, undoItem{
		rel: target.RelPath, passage: passageID, lang: lang,
		version: version, before: before, plain: sp.Plain,
	})
	b.dirty = true
	b.observeSession(target, time.Time{})
	return nil
}

func (b *Book) UndoReplace(target *model.Session) error {
	if target == nil || target.Rich == nil || len(b.undo) == 0 {
		return ErrStale
	}
	item := b.undo[len(b.undo)-1]
	if item.rel != target.RelPath || target.Rich.Version() != item.version {
		return ErrStale
	}
	if !target.Rich.CanUndo() {
		return ErrStale
	}
	target.Rich.Undo()
	g := b.GroupByPath(target.RelPath)
	if g == nil {
		return ErrNoGroup
	}
	p := b.passage(g, item.passage)
	if p.Targets == nil {
		p.Targets = map[string]TargetLink{}
	}
	p.Targets[string(item.lang)] = item.before
	b.putPassage(g, p)
	b.held[string(item.lang)+"/"+item.passage] = item.plain
	b.undo = b.undo[:len(b.undo)-1]
	b.dirty = true
	b.observeSession(target, time.Time{})
	return nil
}

func (b *Book) RestoreGroup(g Group) {
	for i := range b.file.Groups {
		if b.file.Groups[i].ID == g.ID || b.file.Groups[i].Source == g.Source {
			b.file.Groups[i] = g
			b.dirty = true
			return
		}
	}
	b.file.Groups = append(b.file.Groups, g)
	b.dirty = true
}

func spanByID(text richtext.Text, id string) (span, bool) {
	for _, sp := range spansOf(text) {
		if sp.ID == id {
			return sp, true
		}
	}
	return span{}, false
}

func (b *Book) passage(g *Group, id string) PassageLink {
	for _, p := range g.Passages {
		if p.ID == id {
			if p.Targets == nil {
				p.Targets = map[string]TargetLink{}
			}
			return p
		}
	}
	return PassageLink{ID: id, Targets: map[string]TargetLink{}}
}

func (b *Book) putPassage(g *Group, p PassageLink) {
	for i := range g.Passages {
		if g.Passages[i].ID == p.ID {
			g.Passages[i] = p
			return
		}
	}
	g.Passages = append(g.Passages, p)
}

func (b *Book) groupByID(id string) *Group {
	for i := range b.file.Groups {
		if b.file.Groups[i].ID == id {
			return &b.file.Groups[i]
		}
	}
	return nil
}

func newID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf[:])
}

// SyncChoice decides which pane should move so both carets share a passage.
// last is the passage already aligned. The first return is the passage the
// target should adopt; the second is the passage the source should adopt.
// An empty string means no move.
func SyncChoice(on bool, sourceID, targetID, last string) (moveTarget, moveSource, next string) {
	if !on {
		return "", "", sourceID
	}
	if sourceID != "" && sourceID != last {
		return sourceID, "", sourceID
	}
	if targetID != "" && targetID != last && targetID != sourceID {
		return "", targetID, targetID
	}
	if last == "" {
		if sourceID != "" {
			return sourceID, "", sourceID
		}
	}
	return "", "", last
}
