use gpui_kit::base::text::MarkdownExtensions;
use markdown_editor::{BlockEditorModel, BlockId, BlockKind, BlockRef};

pub fn model(source: &str) -> BlockEditorModel {
    BlockEditorModel::from_markdown(source, MarkdownExtensions::default())
        .unwrap_or_else(|err| panic!("parse failed: {err}"))
}

#[allow(dead_code)]
pub fn round_trip(source: &str) -> String {
    model(source).source()
}

#[allow(dead_code)]
pub fn leaves(model: &BlockEditorModel) -> Vec<(BlockId, BlockKind, Option<String>)> {
    model
        .document()
        .leaves()
        .map(|leaf| (leaf.id(), leaf.kind(), leaf.text().map(str::to_owned)))
        .collect()
}

pub fn walk_kind(block: BlockRef<'_>, kind: BlockKind, out: &mut Vec<BlockId>) {
    if block.kind() == kind {
        out.push(block.id());
    }
    for child in block.children() {
        walk_kind(child, kind, out);
    }
}

pub fn blocks_of(model: &BlockEditorModel, kind: BlockKind) -> Vec<BlockId> {
    let mut out = Vec::new();
    walk_kind(model.document().root(), kind, &mut out);
    out
}
