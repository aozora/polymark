//! Composition through GPUI's `EntityInputHandler`, under `TestAppContext`.
//! This drives the editor's IME callbacks. It is not an OS IME.

use std::cell::RefCell;
use std::rc::Rc;

use gpui::{
    AppContext as _, Entity, EntityInputHandler, IntoElement, ParentElement as _, Render, Styled as _,
    TestAppContext, Window, div, px,
};
use markdown_editor::{
    BlockEditor, BlockEditorEvent, BlockEditorState, BlockPosition, BlockSelection,
};

struct Root(Entity<BlockEditorState>);

impl Render for Root {
    fn render(&mut self, _: &mut Window, _: &mut gpui::Context<Self>) -> impl IntoElement {
        div()
            .w(px(640.))
            .h(px(480.))
            .child(BlockEditor::new(&self.0))
    }
}

fn open<'a>(
    cx: &'a mut TestAppContext,
    source: &str,
) -> (Entity<BlockEditorState>, &'a mut gpui::VisualTestContext) {
    cx.update(|cx| {
        gpui_kit::init(cx);
        markdown_editor::init(cx);
    });
    let source = source.to_string();
    let (root, cx) = cx.add_window_view(move |window, cx| {
        Root(cx.new(|cx| BlockEditorState::new(window, cx).with_source(&source)))
    });
    let editor = root.read_with(cx, |root, _| root.0.clone());
    draw(cx);
    (editor, cx)
}

fn place_caret_at_end(editor: &Entity<BlockEditorState>, cx: &mut gpui::VisualTestContext) {
    cx.update(|window, cx| {
        editor.update(cx, |state, cx| {
            let (id, len) = {
                let leaf = state.model().document().leaves().next().unwrap();
                (leaf.id(), leaf.text().map(str::len).unwrap_or(0))
            };
            state
                .set_selection(BlockSelection::caret(BlockPosition::new(id, len)), cx)
                .unwrap();
            state.focus(window, cx);
        });
    });
    draw(cx);
}

fn draw(cx: &mut gpui::VisualTestContext) {
    cx.run_until_parked();
    cx.update(|window, cx| {
        let _ = window.draw(cx);
    });
}

#[gpui::test]
fn preedit_is_marked_and_cancel_does_not_commit(cx: &mut TestAppContext) {
    let (editor, cx) = open(cx, "a");
    place_caret_at_end(&editor, cx);
    let events = Rc::new(RefCell::new(Vec::new()));
    let sink = events.clone();
    cx.update(|_, cx| {
        cx.subscribe(&editor, move |_, event: &BlockEditorEvent, _| {
            sink.borrow_mut().push(format!("{event:?}"));
        })
        .detach();
    });

    cx.update(|window, cx| {
        editor.update(cx, |state, cx| {
            state.replace_and_mark_text_in_range(None, "に", None, window, cx);
            let marked = state.marked_text_range(window, cx);
            assert_eq!(marked, Some(1..2), "UTF-16 length of に is 1");
            assert!(
                state.source().contains('に'),
                "provisional text is inside the document during pre-edit: {}",
                state.source()
            );
            state.replace_and_mark_text_in_range(None, "にほ", None, window, cx);
            assert!(state.marked_text_range(window, cx).is_some());
            state.replace_and_mark_text_in_range(None, "", None, window, cx);
            assert_eq!(state.marked_text_range(window, cx), None);
            assert_eq!(state.source(), "a");
        });
    });
    draw(cx);
    assert_eq!(editor.read_with(cx, |state, _| state.source()), "a");
    let log = events.borrow().join("\n");
    eprintln!("preedit events:\n{log}");
    assert!(
        log.contains("Ime"),
        "pre-edit must be distinguishable from committed typing: {log}"
    );
}

#[gpui::test]
fn commit_is_one_undo_and_reports_caret_bounds(cx: &mut TestAppContext) {
    let (editor, cx) = open(cx, "a");
    place_caret_at_end(&editor, cx);
    cx.update(|window, cx| {
        editor.update(cx, |state, cx| {
            state.replace_and_mark_text_in_range(None, "ni", None, window, cx);
            state.replace_text_in_range(None, "に", window, cx);
            assert_eq!(state.marked_text_range(window, cx), None);
            assert_eq!(state.source(), "aに");
            let bounds = state.bounds_for_range(1..2, Default::default(), window, cx);
            assert!(bounds.is_some(), "caret/pre-edit anchor geometry");
            eprintln!("composition anchor bounds {bounds:?}");
        });
    });
    draw(cx);
    editor.update(cx, |state, cx| state.undo(cx));
    assert_eq!(editor.read_with(cx, |state, _| state.source()), "a");
}
