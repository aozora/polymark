// Package library keeps several documents open in memory so switching
// away does not drop unsaved edits, and writes a recovery snapshot that
// the next launch can reopen.
package library

import (
	"github.com/aozora/polymark/internal/model"
	"github.com/aozora/polymark/internal/store"
)

type Library struct {
	Store   *store.Store
	Open    map[string]*model.Session
	Current string
	Folder  string
}

func Open(dir string) (*Library, error) {
	st, err := store.Open(dir)
	if err != nil {
		return nil, err
	}
	lib := &Library{Store: st, Open: map[string]*model.Session{}}
	recs, err := st.Recoveries()
	if err != nil {
		return nil, err
	}
	for _, r := range recs {
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
		// OpenMarkdown marked the bytes saved. The journal is unsaved work.
		session.ForceDirty(r.SavedDigest)
		lib.Open[r.RelPath] = session
		lib.Current = r.RelPath
	}
	return lib, nil
}

func (l *Library) CreateDocument(title string, lang model.Language) (*model.Session, error) {
	_, session, err := l.Store.CreateDocument(l.Folder, title, lang)
	if err != nil {
		return nil, err
	}
	l.Open[session.RelPath] = session
	l.Current = session.RelPath
	return session, nil
}

func (l *Library) Activate(rel string) (*model.Session, error) {
	if s, ok := l.Open[rel]; ok {
		l.Current = rel
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
	l.Current = rel
	return session, nil
}

func (l *Library) CurrentSession() *model.Session {
	if l.Current == "" {
		return nil
	}
	return l.Open[l.Current]
}

func (l *Library) SaveCurrent() error {
	s := l.CurrentSession()
	if s == nil {
		return model.ErrNoDocument
	}
	return l.Store.Save(s, s.SavedDigest())
}

func (l *Library) Checkpoint() error {
	s := l.CurrentSession()
	if s == nil || !s.Dirty() {
		if s != nil {
			return l.Store.RemoveRecovery(s.ID)
		}
		return nil
	}
	anchor, caret := s.Selection()
	return l.Store.WriteRecovery(store.Recovery{
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
	})
}
