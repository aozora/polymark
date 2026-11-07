# MyGo probe matrix

Separate from `docs/test-matrix.md`. `passed` means this Ubuntu 24.04
environment ran the check and it succeeded. `not-run` was not executed.

| Check | Where | Status |
| --- | --- | --- |
| Module resolves at v0.8.3 with Go 1.27.3 | `probes/mygo-editor` | passed |
| Markdown round trip: headings, bold, italic, lists, links, JP, Cyrillic, Latin, combining dakuten, ZWJ | `TestMarkdownRoundTripKeepsScriptsAndMarks` | passed |
| Pipe tables stay tables | `TestTablesBecomeTabSeparatedParagraphs` | failed as a product requirement. Cells become tab-separated paragraphs. Export logged in `docs/evidence/mygo-test.log` |
| Explicit `Edit` undo restores text and selection | `TestExplicitEditUndoesTextAndSelection` | passed |
| `SetText` clears history | `TestSetTextClearsHistory` | passed |
| Paragraph IDs survive undo | `TestParagraphIDsSurviveUndo` | passed. Markdown does not keep them |
| Editor focus, bold, heading, simulated composition | `TestEditorFormatsAndSimulatedComposition` | passed. `Compose` is the tester, not an OS IME. Pre-edit left `Version` unchanged |
| Caret box during that composition | same test | passed in tester coordinates: `{X:64 Y:41.5 W:1 H:20}` |
| `SetDark` keeps markdown and selection | `TestThemeSwitchKeepsTheDocument` | passed. Not a live desktop theme |
| Native window pixels | `docs/evidence/mygo-editor.png` | passed under Xvfb. Log: CPU fallback, no GPU. GTK 3 present. WebKitGTK not installed and not required for this window |
| OS Japanese IME, candidate window, cancel on device | Devuan, ibus/fcitx | not-run |
| Devuan, sysvinit, i3 | target machine | not-run |
| AppImage | packaging | not-run. MyGo's distribution doc has no AppImage target |
| PDF of the native editor | `Page.PrintToPDF` | not-run. Source returns `errNoPage` for a native-content window |
| docx | source search of v0.8.3 | no API. Only demo filenames |
| Atomic save and crash recovery | `NewDocuments` | not-run. Source and docs describe it |
| Orca, clipboard with another app, Secret Service, HTTP credentials | desktop session | not-run |
