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
one. `-seed` adds `Welcome.md` and an empty Italian linked file only when the
folder has no documents, for a demo. The Italian file is not a translation.
`-theme file.json` reads hex colors and nothing else. A file that starts
with `#!` or has unknown keys is rejected.

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
notes/Hello.it.md             linked target, same sidecar shape, not a translation
.polymark/links.json          schema 1, optional passage links
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

## Linked translations

The source stays on the left. One target language is open on the right.
The other languages stay in memory when you switch, so an unsaved edit is
not dropped. Scroll together is on by default and can be turned off. It
moves the caret to the same passage id. It is not a shared pixel scroll
offset. Both panes have to be in rich text for that move to run.

Passage ids are the rich-text paragraph ids already stored, in order, in
the schema-1 sidecar. Markdown cannot hold them. They are stamped back on
by index when a file is opened. A rich-text edit that keeps the same
paragraphs keeps those ids. Inserting, deleting, or merging paragraphs, or
a Markdown edit that changes the paragraph count, can move which id sits
on a later paragraph. This milestone does not insert or delete paragraphs
in the other language to match. A result whose passage id is not in the
target is rejected. A new source paragraph is recorded as outdated and is
not filled in.

Link state is a new optional `.polymark/links.json` (schema 1). Document
sidecars stay schema 1, so an older project still opens. A build that does
not know about links ignores `links.json` and opens a target file as an
ordinary document. Each target passage records the source revision it was
applied to, its own revision, a text hash, and whether it is protected.
Outdated means the applied source revision is behind the current source
revision. Protected and outdated can both be true. Pending is not stored.
It means a request is in flight in this process. A crash drops in-flight
requests. The window does not enter pending, because it does not send
requests.

A source edit of a paragraph bumps that passage's source revision and
marks every target of it outdated. A manual edit of the target sets
protected and bumps the target revision. Nothing automatic may overwrite
it. Replace passage writes the text you typed, clears protection, and
records one undo step. Undo replace restores the previous characters and
the previous protected and outdated flags, and only if the target document
has not been edited again since that replacement. An empty replacement is
stored as a non-breaking space so the paragraph survives. A normal space
is not used: the Markdown parser trims it and the target paragraphs
collapse into one.

The request boundary is `internal/link`. A request names the group, source
path, passage, target language, and the source and target revisions. A
result is applied only when all of those still match, the session is the
target file, and the passage is not protected. A mismatch is rejected even
if the request was also cancelled. Switching the visible language or the
source document cancels automatic requests that no longer match, and a
late result is still rejected. Only the visible target is eligible for an
automatic request. A hidden language stays outdated until something asks
for it on demand. Tests drive that boundary with a fake result string.
That string is not a translation and is not in the application.

The trigger defaults to a 400ms pause after the source paragraph changes.
The window can switch it to "when leaving a paragraph" or "on save".
Leaving a paragraph is detected when scroll together moves off that
passage. Those triggers only decide when a request would be eligible.
With no provider configured, eligibility does not send anything.

Saving writes the dirty source and the dirty target, then `links.json`.
A checkpoint writes the crash journal first, including the link group, and
then `links.json`. If the process dies after the journal and before
`links.json`, the next launch overlays the journal's group. If it dies
after `links.json` and before the journal, a source revision can move
ahead of the Markdown file. The target is then outdated and is not
overwritten. Recovery still does not restore the editor undo stack.

A pipe table or an HTML table stays in Markdown source and cannot be
linked. Creating a target does not write a file for it. Rich text is still
refused, and saving still writes the original source bytes.

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
- Linked source and one target. Switching Italian and Russian keeps the
  unsaved target. Save, reopen, and crash recovery keep passage ids and
  the protected flag. A schema-1 project with no `links.json` still opens.
  A pipe table and an HTML table stay source-only and are not linked.
- Request tests reject a late result after an edit, an older result
  arriving after a newer one, a different document, a hidden target, a
  cancelled request that has also gone stale, a protected passage, and a
  result aimed at the source document. Explicit replace and its undo
  restore the text and the protected, outdated state.
- The window test shows "Translation is not configured. No text is sent.",
  refuses to change the source language onto an existing target language,
  moves the target caret with scroll together, and does not show a fixture
  translation. Pre-edit of `に` still does not change the committed
  Markdown, the document version, or the source revision.

## Not done

- A real translation engine. No provider is configured. The app makes no
  network request, stores no credential, and spends nothing. Glossary,
  review popover, and engine setup are later.
- Pipe-table editing, PDF, and docx. Deferred. Tables are preserved as
  source and are not linked.
- Rewriting the other language when a paragraph is inserted or deleted.
  Deferred. Results for a missing passage id are rejected.
- Real Japanese IME, candidate window, reconversion. Not run. No ibus or
  fcitx in this environment. The composition test is MyGo's hook, not an
  operating-system input method, and it was not repeated on a Devuan machine.
- Devuan, sysvinit, i3, and a user's machine for this linked-translation
  change. The earlier writing milestone was run there at
  `7388a130abdea2a8569a165ea8f98309a5dc3315`. This change was exercised with
  `go test ./...` and one Xvfb window on the build machine. CPU drawing,
  no GPU. The log line is `mygo: native UI draws without the GPU: there is none`.
  The window shot is `docs/evidence/polymark-linked.png`.
- AppImage and other packaging. Instructions above are the build. MyGo's
  packager has no AppImage target. Live Omarchy theme watch, Orca, and a
  clipboard shared with another process are not in this milestone.
- Undo stack after a crash. Recovery restores the latest unsaved Markdown,
  styles that Markdown can express, the selection, and the link flags in
  the journal. It is not a version history.
- Popup anchors pinned across scroll and resize. The window does not draw
  a popover.
