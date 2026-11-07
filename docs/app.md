# Polymark local writing

This is the first application milestone. The GPUiX and MyGo probes under
`probes/` are unchanged evidence. Nothing here is a release, and no stack
change beyond the already selected Go + native MyGo editor is implied.

## Run

Go 1.27.3 or newer. MyGo v0.8.3, pinned in `go.mod`. Linux needs GTK 3.
This build does not use a webview, so WebKitGTK is not required. The
process uses cgo off. On a machine without a GPU, MyGo draws on the CPU
and logs that.

```sh
export PATH="$HOME/sdk/go/bin:$PATH"   # if the default go is older than 1.27.3
cd polymark
go test ./...
go run ./cmd/polymark -project /path/to/folder
```

`-project` is the folder. If it has no `polymark.json`, the app creates
one. `-seed` adds `Welcome.md` only when the folder has no documents, for
a demo. `-theme file.json` reads hex colors and nothing else. A file that
starts with `#!` or has unknown keys is rejected.

The window that produced `docs/evidence/polymark-window.png` was:

```sh
go build -o /tmp/polymark ./cmd/polymark
# Xvfb, GDK_BACKEND=x11, LIBGL_ALWAYS_SOFTWARE=1
/tmp/polymark -project "$tmpdir" -seed
```

The log line was `mygo: native UI draws without the GPU: there is none`.

## Files

```text
polymark.json                 schema 1, project id and name
notes/Hello.md                Markdown
notes/Hello.md.polymark.json  schema 1, document id, language, fidelity, passage ids
.polymark/recovery/<id>.json  one crash copy of unsaved text, not a history
```

Languages are `en`, `it`, `ru`, and `ja`. Passage ids follow paragraphs in
order. Markdown does not contain them. A source edit that changes the
paragraph count keeps ids for the shared prefix and assigns new ids after
that. `SetText` is not used. Rich undo stays when you open Markdown source
and switch back without editing it. A real source edit returns to rich
text as one `Document.Edit`, so undo restores the previous rich text and
selection.

A file with a pipe table or an HTML `<table` opens in Markdown source.
Rich text is refused, and the banner says why. Saving writes that source.
It is not parsed and exported back as paragraphs.

Saves write a temp file in the same directory, sync it, then rename it.
If the rename is refused, the previous file stays. A save is refused when
the file on disk no longer matches the digest from the last successful
read or write, and when the path is outside the project or is a symlink.

Switching documents keeps the other document's unsaved session in memory.
On each change the recovery file is rewritten. The next launch loads it
dirty. Saving deletes the recovery file.

## Done in this milestone

- Native English window: tree, new folder, new document, language, rich
  text, Markdown source, save.
- Required formatting survives export, mode switch, and reopen in tests:
  headings, bold, italic, lists, links, plus Japanese, Cyrillic, Latin,
  combining dakuten, and a ZWJ emoji in the model test.
- Tester composition: `Compose("に")` does not change the committed
  Markdown or the document version. Cancel then type does not store `に`.
  `TextCaret` returned a box with height, and `VisibleParagraphs` returned
  a box. This is MyGo's test hook, not an operating-system IME.
- Storage tests: conflict does not clobber, a failed rename leaves the
  original and no temp file, symlink write is refused, `..` and slash
  names are refused, recovery reopens unsaved text without having written
  it to the Markdown file.

## Not done

- Pipe-table editing, PDF, and docx. Deferred. Tables are preserved as
  source instead.
- Real Japanese IME, candidate window, reconversion. Not run. No ibus or
  fcitx in this environment.
- Devuan, sysvinit, i3, and a user's machine. The run was Ubuntu 24.04,
  Xvfb, CPU drawing.
- AppImage and other packaging. Instructions above are the build. MyGo's
  packager has no AppImage target.
- Live Omarchy theme watch, Orca, clipboard with another process, dual
  panes, translation, glossary, providers, network, and secrets.
- Undo stack after a crash. Recovery restores the latest unsaved Markdown,
  styles that Markdown can express, and the selection. It is not a version
  history.
- Popup anchors pinned across scroll and resize. Paragraph boxes and the
  caret rectangle are available from the editor APIs used in the test.
  The window does not draw a popover.
