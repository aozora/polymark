# Pinned versions

Recorded 2026-11-07 from the upstream git and registry APIs, then checked
against what this repository resolves. Registry dates are the publisher
timestamps, not a claim that this environment built those artifacts.

| Piece | Pin | License |
| --- | --- | --- |
| Rust | 1.97.1 (`8bab26f4f`, 2026-07-14), same channel as remorses/gpuix and remorses/zed | Rust project licenses |
| Node for the shell | 24.22.2. gpuix-svelte 0.5.0 requires Node >= 24. The image default `node` may be 22. | Node.js license |
| gpuix-svelte | npm `0.5.0` (published 2026-11-06). Docs commit `1bbf51d0a8bab8523ddcc1745610f94c695640db` (2026-11-07) | MIT |
| Bundled Svelte | `5.57.2`, inside the gpuix-svelte 0.5.0 tarball (`package/node_modules/svelte`) | MIT |
| @gpuix/native | npm `0.11.0` (published 2026-10-20). Git tag `@gpuix/native@0.11.0` = `2e31b47ff6432ceebe70e7fc7b0f51a76d7d30cf` | Apache-2.0 |
| gpuix main used by the identity probe | `0ce4f41fca8ffb9c18fd1efd2631f0f4b97a1abf` (2026-11-01). Not published to npm | Apache-2.0 |
| Custom-element registration commit | `2f3ee31c4c40` (2026-10-23), after the 0.11.0 tag | Apache-2.0 |
| remorses/zed submodule at both the 0.11.0 tag and `0ce4f41` | `f77e5cfe9dad04288566f00c66ecf3c58d6f84b3` (2026-10-20) | Apache-2.0 |
| gpui-pre | crates.io `=0.3.11` (2026-11-02). Crate description: snapshot of `zed@f816e1f9263e3a8ef63c4baaa02e1441ce1c6038` (2026-11-01) | Apache-2.0 |
| gpui-kit | crates.io `=0.9.0` (2026-11-02). Git tip at research time `a5bda4aa82318e5f4e67111face6886da3d477c9` is newer than 0.9.0 and is not what the editor resolves | Apache-2.0 |
| gpui-kit-markdown-editor | crates.io `=0.2.0`, git `bb2f1a27203264397378600ac2818d43b05762fc` (2026-11-07) | Apache-2.0 |

`Cargo.lock` and `package-lock.json` are the resolved graphs for the shell
and the contract tests. `probes/gpui-identity/Cargo.lock` is the graph that
produced the `E0308` diagnostic.

## Identity, one line

`gpui-pre` 0.3.11 and gpuix's path dependency on `zed/crates/gpui` at
`f77e5cfe` are not the same crate. Rust treats their `AnyElement` types as
unrelated.
