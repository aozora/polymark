// Package appui is the English native window: a document tree and one
// editor, either rich text or Markdown source.
package appui

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"

	"github.com/aozora/polymark/internal/library"
	"github.com/aozora/polymark/internal/model"
	"github.com/aozora/polymark/internal/store"
	"github.com/aozora/polymark/internal/theme"
)

type App struct {
	Lib    *library.Library
	Name   string
	Banner string
	Status string
	Tokens theme.Tokens
	Boxes  []richtext.ParagraphBox
}

func New(lib *library.Library, tokens theme.Tokens) *App {
	return &App{Lib: lib, Tokens: tokens}
}

func (a *App) View(c *ui.Context) {
	a.refresh()
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		a.tree(c)
		a.editor(c)
	})
}

func (a *App) tree(c *ui.Context) {
	ui.Column(c).Width(240).Padding(8).Gap(6).Children(func() {
		ui.Text(c, "Project").Bold()
		ui.TextInput(c, &a.Name).Placeholder("Name")
		if ui.Button(c, "New document").Clicked() {
			a.createDoc()
		}
		if ui.Button(c, "New folder").Clicked() {
			a.createFolder()
		}
		ui.Scroll(c).Grow(1).Children(func() {
			nodes, err := a.Lib.Store.Tree()
			if err != nil {
				a.Banner = err.Error()
				return
			}
			a.nodes(c, nodes, 0)
		})
	})
}

func (a *App) nodes(c *ui.Context, nodes []store.Node, depth int) {
	for _, n := range nodes {
		label := n.Name
		if depth > 0 {
			label = "  " + label
		}
		if n.Dir {
			label = label + "/"
			if ui.Button(c, label).Clicked() {
				a.Lib.Folder = n.Rel
			}
			a.nodes(c, n.Children, depth+1)
			continue
		}
		if ui.Button(c, label).Clicked() {
			if _, err := a.Lib.Activate(n.Rel); err != nil {
				a.Banner = err.Error()
			} else {
				a.Lib.Folder = parent(n.Rel)
				a.Banner = ""
			}
		}
	}
}

func (a *App) editor(c *ui.Context) {
	ui.Column(c).Grow(1).Padding(12).Gap(8).Children(func() {
		s := a.Lib.CurrentSession()
		if a.Banner != "" {
			ui.Text(c, a.Banner).TextColor(ui.RGB(140, 60, 20))
		}
		if s == nil {
			ui.Text(c, "No document open. Create one in the project tree.")
			return
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).TextColor(ui.RGB(140, 60, 20))
		}
		ui.Row(c).Gap(6).Children(func() {
			for _, lang := range []model.Language{model.LangEnglish, model.LangItalian, model.LangRussian, model.LangJapanese} {
				label := lang.Label()
				btn := ui.Button(c, label)
				if s.Language == lang {
					btn.Background(c.Theme().Accent.Alpha(0.25))
				}
				if btn.Clicked() {
					s.Language = lang
				}
			}
		})
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c, "Rich text").Clicked() {
				if err := s.SwitchMode(model.ModeRich); err != nil {
					a.Banner = err.Error()
				} else {
					a.Banner = ""
				}
			}
			if ui.Button(c, "Markdown").Clicked() {
				if err := s.SwitchMode(model.ModeSource); err != nil {
					a.Banner = err.Error()
				} else {
					a.Banner = ""
				}
			}
			if ui.Button(c, "Save").Clicked() {
				a.save()
			}
		})
		ui.Text(c, a.Status)
		if s.Mode == model.ModeRich && s.Rich != nil {
			opts := richtext.EditorOptions{Label: "Document", Theme: a.Tokens.Rich()}
			ed := richtext.Editor(c, s.Rich, opts).Grow(1)
			if ed.Changed() {
				a.checkpoint(s)
			}
			a.Boxes = s.Rich.VisibleParagraphs()
			return
		}
		area := ui.TextArea(c, &s.Source).Grow(1).Label("Markdown source")
		if area.Composing() {
			a.Status = "Composing. Pre-edit is not saved."
			return
		}
		if area.Changed() {
			anchor, caret := s.Selection()
			s.NoteSource(s.Source, anchor, caret)
			a.checkpoint(s)
		}
	})
}

func (a *App) refresh() {
	s := a.Lib.CurrentSession()
	if s == nil {
		a.Status = "Open a document."
		return
	}
	anchor, caret := s.Selection()
	dirty := ""
	if s.Dirty() {
		dirty = "unsaved"
	} else {
		dirty = "saved"
	}
	a.Status = fmt.Sprintf("%s  %s  %s  %s  selection %d–%d", s.RelPath, s.Language.Label(), s.Mode, dirty, anchor, caret)
}

func (a *App) createDoc() {
	lang := model.LangEnglish
	if s := a.Lib.CurrentSession(); s != nil {
		lang = s.Language
	}
	if _, err := a.Lib.CreateDocument(a.Name, lang); err != nil {
		a.Banner = err.Error()
		return
	}
	a.Name = ""
	a.Banner = ""
}

func (a *App) createFolder() {
	name := a.Name
	if name == "" {
		a.Banner = "Type a folder name first."
		return
	}
	rel, err := a.Lib.Store.CreateFolder(a.Lib.Folder, name)
	if err != nil {
		a.Banner = err.Error()
		return
	}
	a.Lib.Folder = rel
	a.Name = ""
	a.Banner = ""
}

func (a *App) save() {
	s := a.Lib.CurrentSession()
	if s == nil {
		a.Banner = "No document open."
		return
	}
	if err := a.Lib.SaveCurrent(); err != nil {
		a.Banner = err.Error()
		return
	}
	a.Banner = "Saved " + s.RelPath
}

func (a *App) checkpoint(s *model.Session) {
	if err := a.Lib.Checkpoint(); err != nil {
		a.Banner = err.Error()
	}
	_ = s
}

func parent(rel string) string {
	for i := len(rel) - 1; i >= 0; i-- {
		if rel[i] == '/' {
			return rel[:i]
		}
	}
	return ""
}
