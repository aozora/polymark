mod common;

use common::{blocks_of, model};
use markdown_editor::{
    BlockKind, BlockPosition, BlockSelection, CapturedRangeError, EditOrigin,
};

#[test]
fn replace_captured_keeps_undo_and_rejects_stale_ranges() {
    let mut editor = model("before привет after");
    editor.select_end().unwrap();
    let original = editor.selection().clone();
    let revision = editor.revision();
    let mut capture = editor.capture_caret().unwrap();
    editor.replace_captured(&mut capture, " 雪").unwrap();
    assert!(editor.revision() > revision);
    assert_eq!(editor.source_snapshot().revision(), editor.revision());
    assert_eq!(editor.source(), "before привет after 雪");
    assert_ne!(editor.selection(), &original);

    let stale = capture.clone();
    editor.insert_text(" human", EditOrigin::Typing).unwrap();
    assert_eq!(
        editor.replace_captured(&mut capture, "nope"),
        Err(CapturedRangeError::Stale)
    );
    assert!(editor.source().ends_with("human"));
    assert!(editor.undo());
    assert_eq!(editor.source(), "before привет after 雪");
    assert!(!editor.is_capture_current(&stale));
    assert!(editor.undo());
    assert_eq!(editor.source(), "before привет after");
    assert_eq!(editor.selection(), &original);
}

#[test]
fn navigation_and_invalidate_reject_without_changing_text() {
    let mut editor = model("draft");
    editor.select_end().unwrap();
    let mut capture = editor.capture_caret().unwrap();
    let id = editor.document().leaves().next().unwrap().id();
    editor
        .set_selection(BlockSelection::caret(BlockPosition::new(id, 0)))
        .unwrap();
    editor.select_end().unwrap();
    assert_eq!(
        editor.replace_captured(&mut capture, "x"),
        Err(CapturedRangeError::Stale)
    );
    assert_eq!(editor.source(), "draft");

    let capture = editor.capture_caret().unwrap();
    editor.invalidate_captures();
    assert!(!editor.is_capture_current(&capture));
    assert_eq!(editor.source(), "draft");
}

#[test]
fn set_source_drops_history_and_caret() {
    let mut editor = model("keep");
    editor.select_end().unwrap();
    editor.insert_text(" me", EditOrigin::Typing).unwrap();
    assert!(editor.can_undo());
    editor.set_source("replaced 日本語").unwrap();
    assert!(!editor.can_undo());
    assert!(!editor.can_redo());
    assert_eq!(editor.source(), "replaced 日本語");
    let caret = editor.caret().unwrap();
    assert_eq!(caret.offset, 0, "whole-value load parks the caret at the start");
}

#[test]
fn cross_block_text_selection_replaces_one_transaction() {
    let mut editor = model("abc\n\ndef");
    let ids: Vec<_> = editor.document().leaves().map(|leaf| leaf.id()).collect();
    assert_eq!(ids.len(), 2);
    editor
        .set_selection(BlockSelection::Text {
            anchor: BlockPosition::new(ids[0], 1),
            head: BlockPosition::new(ids[1], 2),
        })
        .unwrap();
    let mut capture = editor.capture_selection().unwrap();
    editor.replace_captured(&mut capture, "X").unwrap();
    assert_eq!(editor.source(), "aXf");
    assert!(editor.undo());
    assert_eq!(editor.source(), "abc\n\ndef");
}

#[test]
fn table_cell_selection_is_not_a_text_capture() {
    let mut editor = model("| h1 | h2 |\n| --- | --- |\n| a | b |\n| c | d |");
    let cells = blocks_of(&editor, BlockKind::TableCell);
    assert_eq!(cells.len(), 6, "{}", editor.source());
    editor
        .set_selection(BlockSelection::Cells {
            anchor: cells[2],
            head: cells[5],
        })
        .unwrap();
    assert!(matches!(editor.selection(), BlockSelection::Cells { .. }));
    assert!(matches!(
        editor.capture_selection(),
        Err(CapturedRangeError::NotText)
    ));
    assert!(matches!(
        editor.capture_caret(),
        Err(CapturedRangeError::NotText)
    ));
    assert!(!editor.can_undo());
}

#[test]
fn leaf_range_is_plain_text_not_canonical_markdown_offsets() {
    let mut editor = model("**prefix** !query");
    editor.select_end().unwrap();
    assert_eq!(editor.caret_prefix(), Some("prefix !query"));
    let mut capture = editor.capture_suffix("!query".len()).unwrap();
    editor.replace_captured(&mut capture, "@ok ").unwrap();
    assert_eq!(editor.source(), "**prefix** @ok ");
    assert!(editor.undo());
    assert_eq!(editor.source(), "**prefix** !query");
}
