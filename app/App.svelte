<script lang="ts">
	import { set_css_vars } from 'gpuix-svelte';
	import EditorHost from './EditorHost.svelte';
	import { fallback_tokens, tokens_for, type ThemeName, type ThemeTokens } from './theme.ts';

	const source_markdown = '# Ciao\n\nSide-by-side source slot.';
	const target_markdown = '# Hello\n\nSide-by-side target slot.';

	let theme_name = $state<ThemeName>('fallback');
	let tokens = $derived(tokens_for(theme_name));

	function publish(next: ThemeTokens) {
		set_css_vars({
			bg: next.bg,
			fg: next.fg,
			muted: next.muted,
			accent: next.accent,
			border: next.border,
			selection: next.selection
		});
	}

	// Vars are read when styles are built, which is before effects run.
	publish(fallback_tokens);

	function toggle_theme() {
		theme_name = theme_name === 'fallback' ? 'paper' : 'fallback';
		publish(tokens);
	}

	function on_theme_key(event: { key?: string }) {
		if (event.key === 'Enter' || event.key === ' ') toggle_theme();
	}
</script>

<div class="app">
	<text class="title">Polymark GPUiX feasibility</text>
	<text class="status" testid="status">
		GPUI identity blocks the editor mount. @gpuix/native 0.11.0 has no register_custom_element.
		The panes are hyphenated slots, not a rich-text editor.
	</text>
	<div class="row">
		<EditorHost title="Source" testid="source-editor" source={source_markdown} theme={theme_name} />
		<EditorHost title="Target" testid="target-editor" source={target_markdown} theme={theme_name} />
	</div>
	<div class="btn" testid="theme" role="button" tabindex="0" onclick={toggle_theme} onkeydown={on_theme_key}>
		<text class="btn-label">Theme {theme_name}</text>
	</div>
</div>

<style>
	.app {
		display: flex;
		flex-direction: column;
		width: 100%;
		height: 100%;
		gap: 12px;
		padding: 16px;
		background-color: var(--bg, #1c1b1a);
	}
	.title {
		font-size: 22px;
		color: var(--fg, #ece7df);
	}
	.status {
		font-size: 14px;
		color: var(--muted, #a39e96);
	}
	.row {
		display: flex;
		flex-direction: row;
		flex-grow: 1;
		gap: 12px;
		min-height: 0;
	}
	.btn {
		padding: 8px 12px;
		border-width: 1px;
		border-style: solid;
		border-color: var(--accent, #e6a15c);
		background-color: var(--bg, #1c1b1a);
	}
	.btn-label {
		color: var(--fg, #ece7df);
		font-size: 14px;
	}
</style>
