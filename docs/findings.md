# Findings

Feasibility only. Nothing here is a release, an AppImage, or a Devuan install.

## What was verified from source

These claims were checked against the commits in `docs/versions.md`, not taken
from the candidate README alone.

**GPUiX Svelte 0.5.0 renders a fixed tag set.** `lib/renderer.js` in the
published tarball maps `div`, `text`, `img`, `svg`, `input`, `textarea`,
`anchored`, `code`, `diff`, `markdown`, and `virtual-list`. Any other tag,
including a hyphenated custom element, becomes `div` with a one-time warning.
Props for that rewritten `div` are dropped unless they are in the universal
set (`testId`, `tabIndex`, aria, and a few others). `scripts/patch-hyphenated-tags.mjs`
is the smallest countermeasure: hyphenated names pass through, and
`polymark-editor` is treated as a text input so focus and blur update the
window `editing` flag. The patch refuses to apply if those three functions
drift from the 0.5.0 text.

**Published native 0.11.0 cannot register an outside element.** Tag
`2e31b47` has `CustomElement` and `CustomElementRegistry::register`, and
`with_defaults()` only registers the built-ins. It does not export
`register_custom_element` or `gpui`. Commit `2f3ee31` (2026-10-23) adds the
process-wide registry. Main `0ce4f41` documents it and re-exports `gpui` with
the comment that a second GPUI cannot draw into the GPUIX window. npm `latest`
is still 0.11.0. There is no published tag to pin for that API. gpuix-svelte
0.5.0 depends on `@gpuix/native >=0.11.0 <0.12.0`, so the shell loads the
build that does not have the registry.

**The editor and GPUiX do not share a GPUI crate.** gpui-kit 0.9.0 depends on
`gpui-pre` `=0.3.11`, described as `zed@f816e1f` (2026-11-01). gpuix's
submodule, at both 0.11.0 and main `0ce4f41`, is `remorses/zed@f77e5cfe`
(2026-10-20). Package names differ (`gpui-pre` versus `gpui`), versions differ
(`0.3.11` versus `0.2.2` in the fork's manifest), and the commits differ.
`probes/gpui-identity` is the compile check. A facade that re-exports the
older fork as `gpui-pre` would still have to compile all of gpui-kit and
`gpui-pre-platform` against that fork. That is an unbounded API port, not a
tag adapter. It was not started.

**The editor's host contract matches its public Rust API better than its
README limitations.** `host.rs` at `bb2f1a27` exports `source_snapshot`,
`capture_caret`, `capture_selection`, `capture_suffix`, `capture_leaf_range`,
`replace_captured`, and `invalidate_captures`. Captures are pinned to owner,
document revision, selection epoch, and the expected selection. README says
there is no `source_offset` mapping. The code agrees: leaf ranges are UTF-8
offsets in plain leaf text, not canonical Markdown offsets. README and the
host contract do not describe two different capture APIs; the README is the
shorter limitation and the contract is the implementation.

**Table cell selections are not text captures.** `BlockSelection::Cells` makes
`capture_selection` and `capture_caret` return `NotText`. A rectangle of cells
cannot be replaced through `replace_captured`.

**`set_source` is the destructive whole-value load.** It reparses, bumps the
revision, clears undo and redo, and parks the caret on the first position.
`replace_captured` is one undoable transaction and does not reparse.

**IME composition is an input-handler implementation, not a host event.**
`BlockEditorState` implements `EntityInputHandler::{replace_and_mark_text_in_range, replace_text_in_range, unmark_text, marked_text_range, bounds_for_range}`.
`composition_open` is `pub(crate)`. During pre-edit the marked characters are
inserted into the document, so `source()` contains them, and `marked_text_range`
is `Some`. Commit goes through `replace_text_in_range` and closes the group.
Cancel is an empty `replace_and_mark_text_in_range`, which removes the marked
text. Upstream view tests cover this. A host that translates on every
`source()` change will treat pre-edit as document text unless it also reads
the marked range. This repository cannot see that range from outside the
crate while composition is open, because the marked range is only on the
input-handler methods and `composition_open` is crate-private.

**JS has no composition or selection contract for this editor.** gpuix
`<input>` keeps an internal caret and undo stack (`input.rs`) and does not
emit composition begin, update, commit, or cancel. gpuix-svelte's event
payload has no selection range. `on_selection_change` exists for the window
selection of GPUIX text, which is not the block editor's `BlockSelection`.

**Linux cannot use the headless test renderer.** `lib.rs` at the 0.11.0 tag
says `TestGpuixRenderer` is macOS and Windows only, because wgpu cannot read
a rendered image back on Linux. `GpuixRenderer` can still open a window.
`app/test.ts` exits 2 when `hasTestGpuixRenderer()` is false. That exit is
not a pass.

**Svelte advice is for the bundled compiler.** The running compiler is the
Svelte 5.57.2 bundled in gpuix-svelte, using the custom renderer. Official
`svelte-autofixer` does not load that renderer. Diagnostics about unknown
tags or DOM accessibility are not evidence about GPUI.

**Not verified, and not claimed.** Japanese OS IME, candidate-window
positioning, focus across a live Omarchy theme, AppImage packaging, sysvinit
on Devuan, and any cloud or dictionary service. This environment is Ubuntu
24.04 with tini, Xvfb, and lavapipe available as a Vulkan ICD. It is not
Devuan Excalibur, i3, or a user's machine.

## Runs in this environment

Ubuntu 24.04, Rust 1.97.1, Node 24.22.2, Xvfb, lavapipe. Not Devuan, not i3, no ibus or fcitx.

`cargo test --no-fail-fast` finished with exit 0. Fourteen tests passed: six model round trips, six selection and undo cases, two composition cases. Log: `docs/evidence/contracts.log`. The composition event lines are in `docs/evidence/composition.log`.

Observed normalization, from those tests:

- Japanese, Cyrillic, Latin, U+304B plus combining dakuten U+3099, precomposed U+304C, and ZWJ family emoji survive import and export.
- The serializer appends one extra trailing newline. A second export is stable.
- `a\n\n\n\nb` becomes `a\n\nb`. Empty paragraphs have no Markdown form.
- Leaf positions are UTF-8 bytes. An offset in the middle of the ZWJ sequence is rejected. Replacing the whole sequence with `X` yields `aXb`.
- `SetBlockKind(Heading { level: 2 })` writes `## hello`. Italic typed after `ToggleMark(Italic)` serializes as `<em>!</em>`, not `*!*`. Markdown that was parsed as `*italic*` or `**bold**` still exports with those markers. Command-applied italic and parsed italic are not the same spelling.
- `set_source` drops undo and redo and puts the caret at offset 0. `replace_captured` is one undo step and restores the previous caret.
- A text selection from the middle of one paragraph to the middle of the next replaces both with one string and undoes back to both paragraphs.
- A table cell rectangle selects, and both `capture_selection` and `capture_caret` return `NotText`.
- `capture_suffix` edits the plain leaf (`prefix !query` inside `**prefix** !query`), not the Markdown source offsets.

Composition, through `EntityInputHandler` on a GPUI test window, not through an OS IME:

- The fresh caret is at the start. The tests move it to the end and focus the editor first.
- `replace_and_mark_text_in_range` of `に` then `にほ` makes `marked_text_range` `Some`, and `source()` contains the provisional characters.
- Each of those updates emits `Changed { origin: Ime }` and `SelectionChanged`, and the revision increases. Cancel (empty mark text) emits another `Ime` change and restores `a`. Pre-edit is visible in `source()` and is distinguishable only by `origin: Ime` and the marked range.
- Commit via `replace_text_in_range(None, "に")` clears the mark, leaves `aに`, and one `undo` returns `a`.
- After paint, `bounds_for_range(1..2)` was `Some(Bounds { origin: (9.6px, 7px), size: 1px × 12px })`. That is the test platform's caret box. It is not a positioned OS candidate window.

GPUiX shell:

- `hasTestGpuixRenderer()` is false. `npm run test:headless` exits 2 before `mount_headless`. No headless screenshot was taken.
- `scripts/run-window.sh` opened a real `GpuixRenderer` window under Xvfb (`created native window`, `mount complete`) and wrote `docs/evidence/window-shell.png` (1280×800). gpui logged `Found no xinput mouse pointers`. The picture shows the two slots and the blocker text. It does not show `BlockEditor`, because native 0.11.0 has no factory for `polymark-editor`. Children placed inside that tag did not paint.
- Theme tokens were applied with `set_css_vars` before the first paint and again from the theme control. The screenshot is the fallback palette. A live swap that preserves an editor caret was not run, because the editor is not mounted.

## Tradeoff where the work stops

`scripts/check-gpui-identity.sh` compiled both graphs and rustc rejected the
one-line mount. `cargo_exit=101`. The diagnostic is `E0308`:

```text
expected `gpuix_native::gpui::AnyElement`, found `gpui::AnyElement`
note: there are multiple different versions of crate `gpui` in the dependency graph
zed/crates/gpui/src/element.rs:588          AnyElement  (gpuix, remorses/zed@f77e5cfe)
gpui-pre-0.3.11/src/element.rs:650          AnyElement  (the editor)
```

A Svelte window on the published stack can show chrome and hyphenated slots.
It cannot host `BlockEditor` without either publishing a gpuix native that
exports the registry and rebuilding gpui-kit against that same GPUI, or
forking the editor onto `remorses/zed@f77e5cfe`. Both are larger than a
bridge. The editor's own model tests still run, on `gpui-pre`, and are the
evidence for Markdown, selection, and undo.
