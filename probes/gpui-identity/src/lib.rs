//! Smallest mount of the candidate editor into GPUiX's GPUI.
//!
//! `bridge_element` is the whole adaptation: hand the editor view back as the
//! custom element's paint. It compiles only when `gpui-kit-markdown-editor`
//! and `gpuix-native` share one `gpui` crate. They do not. See
//! `docs/findings.md`.

use gpui::Entity;
use markdown_editor::{BlockEditor, BlockEditorState};

pub fn bridge_element(editor: &Entity<BlockEditorState>) -> gpuix_native::gpui::AnyElement {
    use gpui::IntoElement;
    BlockEditor::new(editor).into_any_element()
}
