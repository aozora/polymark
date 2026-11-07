# Polymark

Local-file writing. The selected editor is Go and native MyGo `ui/richtext`.
The app is `cmd/polymark`. How to build it, the file format, and what is
still untested are in `docs/app.md`.

The GPUiX and MyGo feasibility probes remain under `probes/`. They are not
the app.

The editor under test for the GPUiX route is
[`gpui-kit-markdown-editor`](https://github.com/wabzqem/gpui-kit-markdown-editor)
0.2.0. The shell is
[`gpuix-svelte`](https://github.com/khromov/gpuix-svelte) 0.5.0 on
[`@gpuix/native`](https://github.com/remorses/gpuix) 0.11.0. Those two do not
share a GPUI crate, and the published native build cannot register an external
element. Details, versions, and the test matrix are in `docs/`.

A second, separate probe asks whether [MyGo](https://mygo.egoist.dev/) v0.8.3
can host the writing surface. It lives in `probes/mygo-editor` and is written
up in `docs/mygo-findings.md`. It does not replace the GPUiX results, and it
does not select a stack.

## Application

```sh
export PATH="$HOME/sdk/go/bin:$PATH"   # Go 1.27.3 or newer
go test ./...
go run ./cmd/polymark -project /path/to/folder
```

`-seed` creates `Welcome.md` and an empty Italian linked file when the
folder has no documents. The Italian file is not a translation. `-theme`
reads a JSON file of `#rrggbb` colors and does not run a script.

The window can show the source and one target (`en`, `it`, `ru`, `ja`)
side by side. Passage links, outdated and protected state, and explicit
replacement live in the app. No translation provider is configured, and
the window says so. Details and the limits of this milestone are in
`docs/app.md`.

## Earlier probes

- Headless model tests for Markdown round trip, selection, undo, and the
  difference between `replace_captured` and `set_source`. They link the
  candidate editor and `gpui-pre` 0.3.11. They do not open a GPUiX window.
- A composition test that calls GPUI's input handler. That is the editor's
  IME callback, not an operating-system input method.
- A GPUiX Svelte shell with two hyphenated slots and a token swap. The slots
  are empty of `BlockEditor` until the identity gap closes.
- A compile probe that tries to return the editor's view as a GPUiX element.

## Build

Rust 1.97.1 and Node 24 are required. gpuix-svelte 0.5.0 refuses Node 22.
Linux desktop libraries match gpui-kit's `script/install-linux.sh`: a C
compiler, fontconfig, wayland, `libxkbcommon-x11`, X11, OpenSSL, zstd, alsa,
and Vulkan. A machine without a GPU needs a software Vulkan ICD such as
lavapipe (`mesa-vulkan-drivers`).

```sh
# Node 24 on PATH. The command below is the path used in this environment.
export PATH="$HOME/.nvm/versions/node/v24.22.2/bin:$PATH"

npm ci
npm run test:tags

cargo test -- --test-threads=1
# or, when DISPLAY is unset:
# scripts/run-contracts.sh
```

The Svelte shell:

```sh
npm start
```

`npm run test:headless` exits 2 on Linux. `@gpuix/native` 0.11.0 documents
that `TestGpuixRenderer` exists only for macOS and Windows. That exit is
recorded in `docs/test-matrix.md` and is not a successful headless run.

A real window under Xvfb, when you want pixels from `GpuixRenderer`:

```sh
scripts/run-window.sh
```

The MyGo probe needs Go 1.27.3, which is newer than the image default:

```sh
export PATH="$HOME/sdk/go/bin:$PATH"   # or any Go 1.27.3+
cd probes/mygo-editor
go test -count=1
```

The identity probe clones gpuix and its Zed submodule, then compiles both
GPUI graphs. It is expected to fail rustc. The log is the evidence:

```sh
scripts/check-gpui-identity.sh
```

## Layout

| Path | Role |
| --- | --- |
| `cmd/polymark/` | Native MyGo writing app |
| `internal/` | Project files, document model, window |
| `docs/app.md` | Format, what passed, what was not run |
| `app/` | GPUiX Svelte shell |
| `crates/polymark-editor-contracts/` | Editor contract tests |
| `probes/gpui-identity/` | Separate crate, one-function mount |
| `probes/mygo-editor/` | MyGo v0.8.3 native rich-text probe. Needs Go 1.27.3 |
| `docs/mygo-findings.md` | What MyGo is, and what the probe ran |
| `scripts/patch-hyphenated-tags.mjs` | Install-time bridge for custom tags |
| `docs/findings.md` | What the sources and runs showed |
| `docs/versions.md` | Pins and licenses |
| `docs/test-matrix.md` | Executed and unexecuted checks |

## License

Original Polymark code is MIT (`LICENSE`). Upstream Apache-2.0 and MIT notices
are in `NOTICE` and `docs/licenses/`.
