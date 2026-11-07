// Package library keeps several documents open in memory so switching
// away does not drop unsaved edits, and writes a recovery snapshot that
// the next launch can reopen.
package library

import (
	"encoding/json"
	"time"

	"github.com/aozora/polymark/internal/link"
	"github.com/aozora/polymark/internal/model"
	"github.com/aozora/polymark/internal/store"
)

type Library struct {
	Store   *store.Store
	Links   *link.Book
	Open    map[string]*model.Session
	Current string
	Target  string
	Folder  string
	Visible model.Language
}

func Open(dir string) (*Library, error) {
	st, err := store.Open(dir)
	if err != nil {
		return nil, err
	}
	file, err := st.LoadLinks()
	if err != nil {
		return nil, err
	}
	lib := &Library{Store: st, Links: link.New(), Open: map[string]*model.Session{}}
	lib.Links.Load(file)
	recs, err := st.Recoveries()
	if err != nil {
		return nil, err
	}
	for _, r := range recs {
		if len(r.Group) > 0 {
			var g link.Group
			if json.Unmarshal(r.Group, &g) == nil && g.Source != "" {
				lib.Links.RestoreGroup(g)
			}
		}
		lang, err := model.ParseLanguage(r.Language)
		if err != nil {
			lang = model.LangEnglish
		}
		session := model.OpenMarkdown(r.RelPath, r.Markdown, r.DocID, lang, r.Passages)
		if r.Fidelity == string(model.FidelitySource) {
			session.Fidelity = model.FidelitySource
			session.Mode = model.ModeSource
			session.Source = r.Markdown
			session.Rich = nil
		}
		if r.Mode == string(model.ModeSource) && session.Fidelity == model.FidelityRich {
			_ = session.SwitchMode(model.ModeSource)
		}
		session.SetSelection(r.Anchor, r.Caret)
		session.ForceDirty(r.SavedDigest)
		lib.Open[r.RelPath] = session
		lib.Current = r.RelPath
	}
	if err := lib.settleRecovery(); err != nil {
		return nil, err
	}
	return lib, nil
}

// settleRecovery shows the source beside a recovered target. A target file
// that was the last journal opened is not left as the only current document,
// and a dirty target recovered next to its source stays on screen.
func (l *Library) settleRecovery() error {
	if l.Current == "" {
		return nil
	}
	if g := l.Links.GroupByPath(l.Current); g != nil && l.Current != g.Source {
		lang, _ := l.Links.LanguageOf(l.Current)
		l.Target = l.Current
		l.Visible = lang
		l.Links.SetVisible(lang)
		if _, err := l.ensure(g.Source); err != nil {
			return err
		}
		l.Current = g.Source
		l.Links.SetFocus(g.Source)
		return nil
	}
	l.Links.SetFocus(l.Current)
	g := l.Links.Group(l.Current)
	if g == nil {
		return nil
	}
	for lang, path := range g.Targets {
		s := l.Open[path]
		if s == nil || !s.Dirty() {
			continue
		}
		l.Target = path
		l.Visible = model.Language(lang)
		l.Links.SetVisible(l.Visible)
		break
	}
	return nil
}

func (l *Library) CreateDocument(title string, lang model.Language) (*model.Session, error) {
	_, session, err := l.Store.CreateDocument(l.Folder, title, lang)
	if err != nil {
		return nil, err
	}
	l.Open[session.RelPath] = session
	l.Current = session.RelPath
	l.Target = ""
	l.Links.SetFocus(session.RelPath)
	return session, nil
}

func (l *Library) Focus(rel string) (*model.Session, error) {
	if g := l.Links.GroupByPath(rel); g != nil && rel != g.Source {
		if _, err := l.ensure(g.Source); err != nil {
			return nil, err
		}
		if _, err := l.ensure(rel); err != nil {
			return nil, err
		}
		lang, _ := l.Links.LanguageOf(rel)
		l.Current = g.Source
		l.Target = rel
		l.Visible = lang
		l.Links.SetFocus(g.Source)
		l.Links.SetVisible(lang)
		return l.Open[g.Source], nil
	}
	s, err := l.ensure(rel)
	if err != nil {
		return nil, err
	}
	l.Current = rel
	l.Links.SetFocus(rel)
	l.Target = ""
	if g := l.Links.Group(rel); g != nil && l.Visible != "" {
		if path := g.Targets[string(l.Visible)]; path != "" {
			if _, err := l.ensure(path); err == nil {
				l.Target = path
			}
		}
	}
	return s, nil
}

func (l *Library) CreateTarget(lang model.Language) (*model.Session, error) {
	src := l.CurrentSession()
	if src == nil {
		return nil, model.ErrNoDocument
	}
	l.Links.SetFocus(src.RelPath)
	if path := l.Links.TargetPath(src.RelPath, lang); path != "" {
		s, err := l.ensure(path)
		if err != nil {
			return nil, err
		}
		l.Target = path
		l.Visible = lang
		l.Links.SetVisible(lang)
		return s, nil
	}
	rel, markdown, passages, err := link.PlanTarget(src, lang)
	if err != nil {
		return nil, err
	}
	session := model.OpenMarkdown(rel, markdown, "", lang, passages)
	if err := l.Store.Save(session, ""); err != nil {
		return nil, err
	}
	if err := l.Links.AdoptTarget(src, session); err != nil {
		return nil, err
	}
	if err := l.Store.SaveLinks(l.Links.File()); err != nil {
		return nil, err
	}
	l.Links.MarkSaved()
	l.Open[rel] = session
	l.Target = rel
	l.Visible = lang
	l.Links.SetVisible(lang)
	return session, nil
}

func (l *Library) ShowTarget(lang model.Language) (*model.Session, error) {
	src := l.CurrentSession()
	if src == nil {
		return nil, model.ErrNoDocument
	}
	path := l.Links.TargetPath(src.RelPath, lang)
	if path == "" {
		l.Visible = lang
		l.Links.SetVisible(lang)
		l.Target = ""
		return nil, nil
	}
	s, err := l.ensure(path)
	if err != nil {
		return nil, err
	}
	l.Target = path
	l.Visible = lang
	l.Links.SetFocus(src.RelPath)
	l.Links.SetVisible(lang)
	return s, nil
}

func (l *Library) ensure(rel string) (*model.Session, error) {
	if s, ok := l.Open[rel]; ok {
		return s, nil
	}
	loaded, err := l.Store.Load(rel)
	if err != nil {
		return nil, err
	}
	lang, err := model.ParseLanguage(loaded.Meta.Language)
	if err != nil {
		return nil, err
	}
	session := model.OpenMarkdown(rel, loaded.Markdown, loaded.Meta.ID, lang, loaded.Meta.Passages)
	l.Open[rel] = session
	return session, nil
}

func (l *Library) CurrentSession() *model.Session {
	if l.Current == "" {
		return nil
	}
	return l.Open[l.Current]
}

func (l *Library) TargetSession() *model.Session {
	if l.Target == "" {
		return nil
	}
	return l.Open[l.Target]
}

func (l *Library) SaveEditing() error {
	for _, rel := range []string{l.Current, l.Target} {
		s := l.Open[rel]
		if s == nil || !s.Dirty() {
			continue
		}
		if err := l.Store.Save(s, s.SavedDigest()); err != nil {
			return err
		}
	}
	if l.Links != nil && l.Links.Dirty() {
		if err := l.Store.SaveLinks(l.Links.File()); err != nil {
			return err
		}
		l.Links.MarkSaved()
	}
	return nil
}

func (l *Library) SaveCurrent() error { return l.SaveEditing() }

func (l *Library) Checkpoint() error {
	now := time.Now()
	if l.CurrentSession() != nil {
		l.Links.SetFocus(l.CurrentSession().RelPath)
		l.Links.Observe(l.CurrentSession(), now)
	}
	if l.TargetSession() != nil {
		l.Links.Observe(l.TargetSession(), now)
	}
	var first error
	for _, rel := range []string{l.Current, l.Target} {
		if err := l.checkpointOne(rel); err != nil && first == nil {
			first = err
		}
	}
	if first != nil {
		return first
	}
	if l.Links.Dirty() {
		if err := l.Store.SaveLinks(l.Links.File()); err != nil {
			return err
		}
		l.Links.MarkSaved()
	}
	return nil
}

func (l *Library) checkpointOne(rel string) error {
	s := l.Open[rel]
	if s == nil || !s.Dirty() {
		if s != nil {
			return l.Store.RemoveRecovery(s.ID)
		}
		return nil
	}
	anchor, caret := s.Selection()
	rec := store.Recovery{
		DocID:       s.ID,
		RelPath:     s.RelPath,
		Language:    string(s.Language),
		Mode:        string(s.Mode),
		Fidelity:    string(s.Fidelity),
		Markdown:    s.CommittedMarkdown(),
		Anchor:      anchor,
		Caret:       caret,
		Passages:    s.Passages(),
		SavedDigest: s.SavedDigest(),
		Notice:      s.Notice,
	}
	if g := l.Links.GroupByPath(s.RelPath); g != nil {
		raw, err := json.Marshal(g)
		if err != nil {
			return err
		}
		rec.Group = raw
	}
	return l.Store.WriteRecovery(rec)
}
