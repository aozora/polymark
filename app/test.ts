/**
 * Linux @gpuix/native 0.11.0 builds TestGpuixRenderer as a stub
 * ("wgpu cannot read a rendered image back yet"). This file records that
 * and does not report a skipped renderer as a pass.
 */
import * as native from '@gpuix/native';

const available =
	typeof (native as { hasTestGpuixRenderer?: () => boolean }).hasTestGpuixRenderer === 'function' &&
	(native as { hasTestGpuixRenderer: () => boolean }).hasTestGpuixRenderer();

if (!available) {
	console.error(
		'TestGpuixRenderer is unavailable in this Linux build of @gpuix/native 0.11.0. Headless mount, hit testing, and screenshot() were not run.'
	);
	process.exit(2);
}

const { mount_headless, tree, click_test_id, all_text, screenshot, check, finish, settle } =
	await import('gpuix-svelte/test');
const { default: App } = await import('./App.svelte');

mount_headless(App, { width: 960, height: 640 });
settle();

type Node = { type?: string; testId?: string; children?: Node[] };

function walk(node: Node | null | undefined, out: Node[]) {
	if (!node) return;
	out.push(node);
	for (const child of node.children ?? []) walk(child, out);
}

const nodes: Node[] = [];
walk(tree() as Node, nodes);
check(
	'hyphenated editor tag reaches GPUI',
	nodes.some((node) => node.type === 'polymark-editor')
);
check('status names the blocker', all_text().includes('GPUI identity'));
click_test_id('theme');
settle();
screenshot(new URL('../docs/evidence/headless-shell.png', import.meta.url).pathname);
finish('polymark-shell', 2);
