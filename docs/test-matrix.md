# Test matrix

Status values: `passed` means this environment ran the check and it succeeded.
`failed` means it ran and did not succeed. `not-run` means it was not
executed. A stub that exits before the check is `not-run`, not `passed`.

| Check | Where | Needs | Status |
| --- | --- | --- | --- |
| Hyphenated-tag patch unit test | `npm run test:tags` | Node 24 | passed |
| Markdown fixture round trip (JP, Cyrillic, Latin, combining dakuten, ZWJ) | `cargo test` round_trip | gpui-pre, no window | passed, plus one trailing newline |
| Headings, emphasis, lists, links, tables | `cargo test` round_trip | gpui-pre | passed. Command italic exports as `<em>` |
| Blank-line collapse, then stable | `cargo test` round_trip | gpui-pre | passed (`a\n\n\n\nb` → `a\n\nb`) |
| UTF-8 leaf offsets, mid-codepoint rejected | `cargo test` round_trip | gpui-pre | passed |
| `replace_captured` undo and stale rejection | `cargo test` selection_undo | gpui-pre | passed |
| Navigation and `invalidate_captures` | `cargo test` selection_undo | gpui-pre | passed |
| `set_source` clears history | `cargo test` selection_undo | gpui-pre | passed |
| Cross-block text selection | `cargo test` selection_undo | gpui-pre | passed |
| Table cell selection is `NotText` | `cargo test` selection_undo | gpui-pre | passed |
| Plain-leaf capture versus Markdown source | `cargo test` selection_undo | gpui-pre | passed |
| IME pre-edit / cancel / commit via `EntityInputHandler` | `cargo test` composition | GPUI test window | passed. `Changed { origin: Ime }` fires during pre-edit. Not an OS IME |
| Caret bounds during composition | `cargo test` composition | GPUI test window | passed on the test platform: origin `(9.6px, 7px)`, size `1×12 px` |
| OS Japanese IME, candidate window, mixed-script focus | manual on Devuan | ibus/fcitx, fonts, GPU | not-run |
| Headless GPUiX `mount_headless` / `screenshot()` | `npm run test:headless` | TestGpuixRenderer | not-run. Linux stub, process exit 2 |
| Real GPUiX window pixels | `scripts/run-window.sh` | X11, Vulkan | passed under Xvfb and lavapipe. See `docs/evidence/window-shell.png`. No xinput pointer. Editor not mounted |
| Single GPUI identity compile | `scripts/check-gpui-identity.sh` | both GPUI graphs | failed, and that failure is the result. rustc E0308, two `AnyElement` types. `cargo_exit=101` |
| AppImage / sysvinit / no systemd | packaging | Devuan root, linuxdeploy | not-run |
| Live theme preserves editor focus | future bridge | mounted editor | not-run |

The command log for the last cargo run is `docs/evidence/contracts.log` when
that script has been executed. Identity output is
`docs/evidence/gpui-identity-check.log`. The window attempt is
`docs/evidence/window-shell.log`.
