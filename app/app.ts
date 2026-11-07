import { render_hot } from 'gpuix-svelte';

render_hot(new URL('./App.svelte', import.meta.url), {
	title: 'Polymark GPUiX feasibility',
	width: 960,
	height: 640
});
