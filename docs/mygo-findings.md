# MyGo feasibility

This note is separate from `docs/findings.md`. The GPUiX probe, its tests, and
its screenshot stay as they were. Nothing here selects a stack.

## What MyGo is

Primary docs: <https://mygo.egoist.dev/> and
<https://mygo.egoist.dev/docs/ui>. Source:
<https://github.com/egoist/mygo>. Pin used here: tag **v0.8.3**, commit
`5add3817c74bb7505d0b473bc5fb28abe2201b99`, published 2026-11-07. `go.mod`
requires **Go 1.27.3**. License: **MIT** (`LICENSE` in that tag).

MyGo is one Go module with two window kinds:

- **Native UI** (`package ui`). The app writes Go. MyGo draws frames itself.
  Docs say Metal on macOS, Direct3D 11 on Windows, and OpenGL on Linux.
  There is no HTML and no webview in that window. Linux loads **GTK 3** and
  OpenGL at run time with [purego](https://github.com/ebitengine/purego).
  `internal/linux/ffi.go` at v0.8.3 says the Linux backend is pure Go and
  does not use cgo. `mygo build` defaults to `CGO_ENABLED=0`.
- **Web frontend.** The OS webview: WKWebView, **WebKitGTK 4.1** on Linux,
  WebView2 on Windows. Any web UI can sit there and call Go through generated
  TypeScript. A native window can also embed a web view (`ui` web view).

This environment's Go was 1.22.2. The probe used a local Go 1.27.3 toolchain
(`~/sdk/go`), not a change to the system default.

`internal/linux` has no `systemd` references. Packaging in
`docs/distribution.md` at v0.8.3 emits a binary, a `.desktop` file, a
`.tar.gz` with `install.sh`, a Debian package, and an Arch `PKGBUILD`.
That page does not mention an AppImage. X11 is named for taskbar grouping;
Wayland is named for matching the desktop entry. i3 was not run.

Native UI's documented runtime library is GTK 3. Web pages add WebKitGTK
4.1. This machine has GTK 3, libGL, and libsecret. It does not have
`libwebkit2gtk` on the loader path. The native probe started without it.

The window log was: `mygo: native UI draws without the GPU: there is none`.
The process still opened a window and painted on the CPU under Xvfb and
lavapipe. That is a software path, not a measured frame time.

## Editor path

The native rich-text editor is `github.com/egoist/mygo/ui/richtext` at the
same tag. Docs: <https://mygo.egoist.dev/docs/ui/rich-text-editor>. It is
attributed text (paragraphs and runs), not a browser `contenteditable`.
`Text.Markdown` and `ParseMarkdown` are the Markdown boundary. The editor
surface is `richtext.Editor`.

`ui/markdown` is the read-only document renderer. It is not the editor.
The code editor (`ui` code editor) is a code surface, not the writing
surface used here.

Probe: `probes/mygo-editor`, module `github.com/aozora/polymark/probes/mygo-editor`,
dependency `github.com/egoist/mygo v0.8.3`. `go test -count=1` passed
(log `docs/evidence/mygo-test.log`).

Observed on that pin:

- Headings, bold, italic, bullet lists, and links round-trip through
  `ParseMarkdown` / `Markdown` for the multilingual fixture. Japanese,
  Cyrillic, Latin, U+304B plus combining dakuten U+3099, and a ZWJ family
  emoji stayed in the text or the Markdown.
- A pipe table does not stay a table. `ParseMarkdown` of
  `| 語 | б |` / `| 漢 | 👨‍👩‍👧 |` stores rows as paragraphs whose cells are
  separated by tabs, with the header bold. `Markdown()` wrote:

  ```markdown
  **語	б**

  漢	👨‍👩‍👧
  ```

  The comment in `ui/richtext/markdown_parse.go` says the same thing.
  Alignment and column structure are not in the model.
- Offsets on `Document` are rune indexes, newlines included. That is the
  opposite counting from the GPUiX candidate's UTF-8 byte leaf offsets.
- `Document.Edit` is one undo step. Undo restored both the text and the
  selection from before the edit.
- `SetText` clears undo and redo and moves the selection to the end.
  That is the destructive whole-value replacement.
- `IdentifyParagraphs` assigns paragraph IDs. They came back with undo.
  Markdown export does not keep them (the rich-text docs say clipboard,
  HTML, Markdown, and RTF omit IDs).
- `ui.Tester.Compose` drives the editor's input-method client. During
  `Compose("に", 1)` the document string stayed `hello\nworld` and
  `Version` stayed 3. Cancel (`Compose("", 0)`) left that string.
  `Type("に")` committed, and `Undo` removed the commit. Caret geometry
  from `Tester.TextCaret` during the pre-edit was
  `{X:64 Y:41.5 W:1 H:20}` in tester coordinates. The source comment in
  `ui/richtext/ime.go` says the composition is a virtual insertion until
  commit. **This is the tester's composition event, not an OS IME.**
  No ibus or fcitx session ran. Candidate-window placement on a real
  input method was not observed.
- `Tester.SetDark(true)` then `SetDark(false)` left the Markdown and the
  selection unchanged. That is the tester's preference bit. A live
  desktop theme file, Omarchy, or i3 was not watched.

The window binary `go build -o /tmp/polymark-mygo` ran under Xvfb
(`GDK_BACKEND=x11`, `LIBGL_ALWAYS_SOFTWARE=1`). Screenshot:
`docs/evidence/mygo-editor.png`. It shows the native editor with a heading,
bold, italic, Japanese, and Cyrillic. The log is
`docs/evidence/mygo-window.log`. WenQuanYi Micro Hei is installed and
日本語 is visible. A color-emoji font for the ZWJ sequence was not
established; the screenshot's emoji slot is not evidence of emoji
shaping.

## Other product constraints

- **Files.** `mygo.NewDocuments` (docs `docs/documents.md`) reads and writes
  on a goroutine, autosaves, keeps a recovery copy under `PathUserData`,
  and refuses a save when the file stamp changed (`ErrDocumentChanged`).
  `document_file.go` compares modification time and size. The probe did
  not run a crash recovery.
- **PDF.** `Page.PrintToPDF` prints a **web** page through the platform
  web view. It returns an error for a native-content window (`errNoPage`
  in `pdf.go`). Native rich text has no PDF exporter in this pin.
- **docx.** No importer or exporter. The only `.docx` hits are demo file
  names in `website/docs-demo/collections.go`.
- **Clipboard.** The rich-text editor copies its own JSON format, HTML,
  and plain text, and pastes the richest available, including RTF and
  Markdown. Not exercised against another process here.
- **Accessibility.** Docs say the editor is a text field and that Orca
  reads ATK run attributes. Not run against Orca.
- **Secrets and HTTP.** `mygo.Secrets` uses the login keychain, Windows
  Credential Manager, or Linux **libsecret** / Secret Service. Without a
  store it returns `errors.ErrUnsupported`. On Linux, other apps of the
  same user may read the secrets without a prompt (comment in
  `secrets.go`). Go's `net/http` is available to an app; MyGo also has
  network helpers. None of that was called. Cloud translation stays
  opt-in and was not tested.
- **Copyleft.** MyGo and this probe's original code are MIT. GTK 3 and,
  if a web view is opened, WebKitGTK are LGPL libraries loaded at run
  time, not compiled into the Go binary by cgo in this configuration.
  libsecret is the same kind of runtime dependency for `Secrets`. Shipping
  a static build of those libraries would change the obligation; this
  probe did not do that.

## Recommendation

MyGo's native `ui/richtext` editor is a feasible writing surface to review:
one MIT module, a Go 1.27.3 build with cgo off, a real attributed editor,
Markdown import and export, undoable edits, and pre-edit text that stayed
out of the document in the tester. The GPUiX mount is still the recorded
`E0308` in `docs/evidence/gpui-identity-check.log`.

The gaps that matter before any selection are pipe-table fidelity, PDF and
docx from the native document, an AppImage (not in MyGo's packager), a
real Japanese IME on Devuan/X11, and GTK 3 as a runtime dependency even
though the Go build itself is cgo-free. No stack change is made by this
branch.
