<script lang="ts">
	import type { ThemeName } from './theme.ts';

	type Props = {
		title: string;
		testid: string;
		source: string;
		theme: ThemeName;
	};

	let { title, testid, source, theme }: Props = $props();
	let slot_label = $derived(`${title} slot has no BlockEditor factory in @gpuix/native 0.11.0`);
</script>

<div class="pane">
	<text class="label">{title}</text>
	<text class="note">native element slot · theme {theme}</text>
	<polymark-editor class="editor" {source} tabindex="0" {testid} role="textbox" aria-label={title}>
		<text class="slot">{slot_label}</text>
	</polymark-editor>
</div>

<style>
	.pane {
		display: flex;
		flex-direction: column;
		flex-grow: 1;
		flex-basis: 0;
		min-width: 0;
		gap: 8px;
		padding: 12px;
		border-width: 1px;
		border-style: solid;
		border-color: var(--border, #3a3835);
		background-color: var(--bg, #1c1b1a);
	}
	.label {
		font-size: 18px;
		color: var(--fg, #ece7df);
	}
	.note,
	.slot {
		font-size: 13px;
		color: var(--muted, #a39e96);
	}
	.editor {
		display: flex;
		flex-direction: column;
		flex-grow: 1;
		min-height: 180px;
		padding: 8px;
		border-width: 1px;
		border-style: solid;
		border-color: var(--accent, #e6a15c);
	}
</style>
