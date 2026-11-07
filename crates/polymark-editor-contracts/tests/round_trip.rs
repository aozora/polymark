mod common;

use common::{blocks_of, leaves, model, round_trip};
use markdown_editor::{BlockKind, BlockPosition, BlockSelection, EditorCommand, MarkKind};

const FIXTURE: &str = include_str!("fixtures/multilingual.md");

#[test]
fn multilingual_fixture_round_trips_byte_for_byte() {
    let once = round_trip(FIXTURE);
    let twice = round_trip(&once);
    assert_eq!(twice, once, "second serialize must be stable");
    assert_eq!(
        once,
        format!("{FIXTURE}\n"),
        "the only export change is one extra trailing newline"
    );
}

#[test]
fn scripts_and_emoji_keep_leaf_text() {
    let editor = model(FIXTURE);
    let text = leaves(&editor)
        .into_iter()
        .filter_map(|(_, _, text)| text)
        .collect::<Vec<_>>()
        .join("\n");
    assert!(text.contains("привет"), "{text}");
    assert!(text.contains("日本語"), "{text}");
    assert!(text.contains("👨‍👩‍👧‍👦"), "{text}");
    assert!(text.contains("👨‍👩‍👧"), "{text}");
    assert!(text.contains('漢'), "{text}");
    assert!(text.contains("か\u{3099}"), "combining dakuten was composed or dropped: {text}");
    assert!(text.contains('が'), "{text}");
}

#[test]
fn headings_lists_emphasis_links_and_tables_survive_parse() {
    let editor = model(FIXTURE);
    let kinds: Vec<_> = leaves(&editor).into_iter().map(|(_, kind, _)| kind).collect();
    assert!(
        kinds.contains(&BlockKind::Heading { level: 1 }),
        "{kinds:?}"
    );
    assert!(kinds.contains(&BlockKind::Paragraph), "{kinds:?}");
    assert!(kinds.contains(&BlockKind::TableCell), "{kinds:?}");

    let items = blocks_of(&editor, BlockKind::ListItem { checked: None });
    assert_eq!(items.len(), 3, "three bullets");
    let lists = blocks_of(&editor, BlockKind::List { ordered: false });
    assert_eq!(lists.len(), 1);

    let mut bold = model("say **ciao** please");
    let id = bold.document().leaves().next().unwrap().id();
    bold.set_selection(BlockSelection::Text {
        anchor: BlockPosition::new(id, 4),
        head: BlockPosition::new(id, 8),
    })
    .unwrap();
    assert_eq!(bold.mark_state(MarkKind::Bold), markdown_editor::MarkState::On);

    let link = model("[ciao](https://example.test/путь)");
    let id = link.document().leaves().next().unwrap().id();
    let mut link = link;
    link.set_selection(BlockSelection::caret(BlockPosition::new(id, 1)))
        .unwrap();
    assert!(link.active_marks().has(MarkKind::Link));
    assert_eq!(
        link.active_marks().link_url().map(|url| url.to_string()),
        Some("https://example.test/путь".to_string())
    );
}

#[test]
fn commands_build_heading_emphasis_list_and_table() {
    let mut editor = model("hello");
    editor
        .run(EditorCommand::SetBlockKind(BlockKind::Heading { level: 2 }))
        .unwrap();
    assert_eq!(editor.source(), "## hello");
    editor.select_end().unwrap();
    editor
        .run(EditorCommand::ToggleMark(MarkKind::Italic))
        .unwrap();
    editor
        .insert_text("!", markdown_editor::EditOrigin::Typing)
        .unwrap();
    // A mark applied by command serializes as HTML, not as `*!*`.
    assert_eq!(editor.source(), "## hello<em>!</em>");
    editor.undo();
    editor
        .run(EditorCommand::ToggleList { ordered: false })
        .unwrap();
    assert!(editor.source().starts_with("- "), "{:?}", editor.source());

    let mut table = model("");
    table
        .run(EditorCommand::InsertTable { rows: 2, cols: 2 })
        .unwrap();
    let cells = blocks_of(&table, BlockKind::TableCell);
    assert_eq!(cells.len(), 4, "{}", table.source());
    assert!(table.source().contains('|'), "{}", table.source());
}

#[test]
fn extra_blank_lines_collapse_and_then_stay_stable() {
    let once = round_trip("a\n\n\n\nb");
    let twice = round_trip(&once);
    assert_eq!(twice, once, "second save must be stable, got {once:?}");
    assert_eq!(
        once, "a\n\nb",
        "empty paragraphs have no Markdown form; record the actual collapse"
    );
}

#[test]
fn codepoint_and_zwj_offsets_are_utf8_bytes() {
    let source = "a👨‍👩‍👧‍👦b";
    let editor = model(source);
    let leaf = editor.document().leaves().next().unwrap();
    let text = leaf.text().unwrap().to_owned();
    let id = leaf.id();
    assert_eq!(text, source);
    let start = "a".len();
    let end = text.find('b').unwrap();
    assert!(end > start + 4, "ZWJ family is more than one scalar: {end}");
    let mut editor = editor;
    editor
        .set_selection(BlockSelection::Text {
            anchor: BlockPosition::new(id, start),
            head: BlockPosition::new(id, end),
        })
        .unwrap();
    let mut capture = editor.capture_selection().unwrap();
    editor.replace_captured(&mut capture, "X").unwrap();
    assert_eq!(editor.document().leaves().next().unwrap().text(), Some("aXb"));

    let mut mid = model(source);
    let id = mid.document().leaves().next().unwrap().id();
    let err = mid.set_selection(BlockSelection::caret(BlockPosition::new(id, start + 1)));
    assert!(err.is_err(), "a mid-codepoint UTF-8 offset must be rejected");
}
