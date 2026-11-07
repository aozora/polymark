package main

import (
	"flag"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"

	"github.com/aozora/polymark/internal/appui"
	"github.com/aozora/polymark/internal/library"
	"github.com/aozora/polymark/internal/model"
	"github.com/aozora/polymark/internal/theme"
)

func main() {
	project := flag.String("project", "", "project folder")
	themePath := flag.String("theme", "", "optional JSON color file, data only")
	seed := flag.Bool("seed", false, "create one example document when the project is empty")
	flag.Parse()
	if *project == "" {
		log.Fatal("usage: polymark -project DIR")
	}
	lib, err := library.Open(*project)
	if err != nil {
		log.Fatal(err)
	}
	if *seed {
		nodes, err := lib.Store.Tree()
		if err != nil {
			log.Fatal(err)
		}
		if len(nodes) == 0 {
			if _, err := lib.CreateDocument("Welcome", model.LangEnglish); err != nil {
				log.Fatal(err)
			}
			s := lib.CurrentSession()
			extra := richtext.ParseMarkdown("**Bold**, *italic*, and a [link](https://example.test).\n\n- one\n- 日本語\n- привет\n")
			s.Rich.Edit(func(e *richtext.Edit) {
				n := e.Text().Len()
				e.ReplaceString(n, n, "\n", richtext.Attributes{})
				e.Replace(e.Text().Len(), e.Text().Len(), extra)
			})
			s.SetSelection(0, 0)
			if err := lib.SaveCurrent(); err != nil {
				log.Fatal(err)
			}
			if _, err := lib.CreateTarget(model.LangItalian); err != nil {
				log.Fatal(err)
			}
		}
	}
	var tokens theme.Tokens
	if *themePath != "" {
		tokens, err = theme.Load(*themePath)
		if err != nil {
			log.Fatal(err)
		}
	}
	app := appui.New(lib, tokens)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Polymark",
			Width:   1280,
			Height:  800,
			Content: ui.View(app.View),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
