// Package appui is the English native window: a document tree, the source
// editor, and one linked target. Translation providers are not configured.
package appui

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"

	"github.com/aozora/polymark/internal/library"
	"github.com/aozora/polymark/internal/link"
	"github.com/aozora/polymark/internal/model"
	"github.com/aozora/polymark/internal/store"
	"github.com/aozora/polymark/internal/theme"
)

type App struct {
	Lib     *library.Library
	Name    string
	Replace string
	Banner  string
	Status  string
	Tokens  theme.Tokens
	Boxes   []richtext.ParagraphBox
	Sync    bool
	lastSrc string
	lastTgt string
	synced  bool
}

func New(lib *library.Library, tokens theme.Tokens) *App {
	a := &App{Lib: lib, Tokens: tokens, Sync: true}
	if lib != nil && lib.Links != nil {
		a.Sync = lib.Links.ScrollTogether()
	}
	return a
}

func (a *App) View(c *ui.Context) {
	a.align()
	a.refresh()
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		a.tree(c)
		a.editor(c)
		a.target(c)
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
			if _, err := a.Lib.Focus(n.Rel); err != nil {
				a.Banner = err.Error()
			} else {
				a.Lib.Folder = parent(n.Rel)
				a.synced = false
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
					if a.Lib.Links != nil && a.Lib.Links.TargetPath(s.RelPath, lang) != "" {
						a.Banner = lang.Label() + " is already a linked target. The source language was left unchanged."
					} else {
						s.Language = lang
						a.Banner = ""
					}
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

func (a *App) target(c *ui.Context) {
	ui.Column(c).Grow(1).Padding(12).Gap(8).Children(func() {
		ui.Text(c, "Translation").Bold()
		ui.Text(c, "Translation is not configured. No text is sent.")
		src := a.Lib.CurrentSession()
		ui.Row(c).Gap(6).Children(func() {
			label := "Scroll together: off"
			if a.Sync {
				label = "Scroll together: on"
			}
			if ui.Button(c, label).Clicked() {
				a.Sync = !a.Sync
				a.synced = false
				if a.Lib.Links != nil {
					a.Lib.Links.SetScrollTogether(a.Sync)
				}
			}
			trig := "After a pause"
			if a.Lib.Links != nil {
				switch a.Lib.Links.Trigger() {
				case link.TriggerLeave:
					trig = "When leaving a paragraph"
				case link.TriggerSave:
					trig = "On save"
				}
			}
			if ui.Button(c, trig).Clicked() && a.Lib.Links != nil {
				switch a.Lib.Links.Trigger() {
				case link.TriggerPause:
					a.Lib.Links.SetTrigger(link.TriggerLeave)
				case link.TriggerLeave:
					a.Lib.Links.SetTrigger(link.TriggerSave)
				default:
					a.Lib.Links.SetTrigger(link.TriggerPause)
				}
			}
		})
		if src == nil {
			ui.Text(c, "Open a source document to link a translation.")
			return
		}
		ui.Row(c).Gap(6).Children(func() {
			for _, lang := range []model.Language{model.LangEnglish, model.LangItalian, model.LangRussian, model.LangJapanese} {
				if lang == src.Language {
					continue
				}
				label := "Show " + lang.Label()
				if a.Lib.Visible == lang {
					label = lang.Label() + " target"
				}
				if ui.Button(c, label).Clicked() {
					a.showTarget(lang)
				}
			}
		})
		tgt := a.Lib.TargetSession()
		if tgt == nil {
			ui.Text(c, "No target document yet. Showing a language creates an empty linked file. It is not a translation.")
			return
		}
		if id, _, ok := link.PassageAt(tgt, func() int { _, c := tgt.Selection(); return c }()); ok {
			ui.Text(c, tgt.Language.Label()+" passage "+a.Lib.Links.Status(src.RelPath, id, tgt.Language))
		}
		ui.TextInput(c, &a.Replace).Placeholder("Replacement text")
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c, "Replace passage").Clicked() {
				a.replacePassage()
			}
			if ui.Button(c, "Undo replace").Clicked() {
				if err := a.Lib.Links.UndoReplace(tgt); err != nil {
					a.Banner = err.Error()
				} else {
					a.Banner = "Restored the passage and its protection state."
					a.checkpoint(tgt)
				}
			}
		})
		if tgt.Notice != "" {
			ui.Text(c, tgt.Notice).TextColor(ui.RGB(140, 60, 20))
		}
		if tgt.Mode == model.ModeRich && tgt.Rich != nil {
			ed := richtext.Editor(c, tgt.Rich, richtext.EditorOptions{Label: "Translation", Theme: a.Tokens.Rich()}).Grow(1)
			if ed.Changed() {
				a.checkpoint(tgt)
			}
			return
		}
		area := ui.TextArea(c, &tgt.Source).Grow(1).Label("Translation source")
		if area.Composing() {
			return
		}
		if area.Changed() {
			tgt.NoteSource(tgt.Source, 0, 0)
			a.checkpoint(tgt)
		}
	})
}

func (a *App) showTarget(lang model.Language) {
	if a.Lib.Links.TargetPath(a.Lib.Current, lang) == "" {
		if _, err := a.Lib.CreateTarget(lang); err != nil {
			a.Banner = err.Error()
			return
		}
		a.Banner = lang.Label() + " file created. Translation is not configured."
	} else if _, err := a.Lib.ShowTarget(lang); err != nil {
		a.Banner = err.Error()
		return
	}
	a.synced = false
}

func (a *App) replacePassage() {
	tgt := a.Lib.TargetSession()
	src := a.Lib.CurrentSession()
	if tgt == nil || src == nil {
		a.Banner = "No linked target."
		return
	}
	if a.Replace == "" {
		a.Banner = "Type the replacement first. Nothing was sent anywhere."
		return
	}
	_, caret := tgt.Selection()
	id, _, ok := link.PassageAt(tgt, caret)
	if !ok {
		a.Banner = "The target caret is not in a linked passage."
		return
	}
	if err := a.Lib.Links.Replace(tgt, id, a.Replace); err != nil {
		a.Banner = err.Error()
		return
	}
	a.Replace = ""
	a.Banner = "Replaced the linked passage. Undo replace restores the text and its protection."
	a.checkpoint(tgt)
}

func (a *App) align() {
	if a.Lib == nil || !a.Sync {
		return
	}
	src := a.Lib.CurrentSession()
	tgt := a.Lib.TargetSession()
	if src == nil || tgt == nil {
		return
	}
	_, srcCaret := src.Selection()
	_, tgtCaret := tgt.Selection()
	srcID, _, _ := link.PassageAt(src, srcCaret)
	tgtID, _, _ := link.PassageAt(tgt, tgtCaret)
	if a.synced && srcID != a.lastSrc && a.lastSrc != "" && a.Lib.Links != nil {
		a.Lib.Links.MarkLeft(a.lastSrc)
	}
	if !a.synced {
		a.lastSrc, a.lastTgt = srcID, tgtID
		a.synced = true
		return
	}
	moveT, moveS, next := link.SyncChoice(true, srcID, tgtID, a.lastSrc)
	if moveT != "" {
		if start, ok := link.PassageStart(tgt, moveT); ok {
			tgt.SetSelection(start, start)
			a.lastSrc, a.lastTgt = next, next
		}
	} else if moveS != "" {
		if start, ok := link.PassageStart(src, moveS); ok {
			src.SetSelection(start, start)
			a.lastSrc, a.lastTgt = next, next
		}
	}
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
	if a.Lib.Links != nil && a.Lib.Links.Trigger() == link.TriggerSave {
		a.Lib.Links.NoteSaved()
	}
	if err := a.Lib.SaveEditing(); err != nil {
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
