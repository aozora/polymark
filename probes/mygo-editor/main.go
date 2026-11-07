// A native MyGo window showing the same rich-text editor the tests drive.
// It is a feasibility probe, not the Polymark app.
package main

import (
	"log"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

func main() {
	source := "# Polymark\n\nMyGo rich text. 日本語, привет, か\u3099, 👨‍👩‍👧‍👦.\n\n**bold** and *italic*.\n"
	if len(os.Args) > 1 {
		b, err := os.ReadFile(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		source = string(b)
	}
	doc := richtext.NewDocument(richtext.ParseMarkdown(source))
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:  "Polymark MyGo probe",
			Width:  720,
			Height: 480,
			Content: ui.View(func(c *ui.Context) {
				ui.Column(c).Fill().Padding(16).Gap(8).Children(func() {
					ui.Text(c, "Native rich text. Not a webview.").TextColor(c.Theme().TextMuted)
					richtext.Editor(c, doc).Grow(1)
				})
			}),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
