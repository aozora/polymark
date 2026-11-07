# GPUI identity probe

This crate is not part of the Polymark workspace. `cargo check` here is the
compiled-compatibility test for mounting `gpui-kit-markdown-editor` 0.2.0 in
a GPUiX custom element.

Expected result: rustc rejects `bridge_element` because `BlockEditor` paints
with `gpui-pre` 0.3.11 and `gpuix-native` (git `0ce4f41`) paints with
`remorses/zed` commit `f77e5cfe`. Those are different crate identities, so
their `AnyElement` types are not the same type.

Published `@gpuix/native` 0.11.0 (`2e31b47`) does not export `gpui` or
`register_custom_element`, so the probe cannot target the npm binary's
sources and still name `gpuix_native::gpui`.

Run `scripts/check-gpui-identity.sh` from the repository root. A non-zero
exit is the recorded result, not a failed product build.
