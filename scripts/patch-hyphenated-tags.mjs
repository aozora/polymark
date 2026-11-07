/**
 * Let gpuix-svelte 0.5.0 pass hyphenated tags through to native.
 *
 * Published renderer.js maps every unknown tag to `div` and drops the
 * warning once. GPUIX custom elements are required to contain `-`, so this
 * patch forwards those names and treats `polymark-editor` as a text input
 * for focus tracking. Built-in names are unchanged.
 *
 * The patch refuses to run if the installed renderer does not match the
 * pinned 0.5.0 source. It does not vendor the renderer.
 */
import { readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const MARKER = 'polymark-hyphenated-custom-tags';

const MAP_TAG = `function map_tag(name) {
    if (GPUI_TAGS.has(name))
        return name;
    if (!warned_tags.has(name)) {
        warned_tags.add(name);
        console.warn(\`[gpuix-svelte] <\${name}> has no GPUI equivalent — rendering it as <div>.\`);
    }
    return 'div';
}`;

const MAP_TAG_PATCHED = `function map_tag(name) {
    if (GPUI_TAGS.has(name))
        return name;
    // ${MARKER}: GPUIX custom elements contain "-". Forward them.
    if (name.includes('-'))
        return name;
    if (!warned_tags.has(name)) {
        warned_tags.add(name);
        console.warn(\`[gpuix-svelte] <\${name}> has no GPUI equivalent — rendering it as <div>.\`);
    }
    return 'div';
}`;

const INTERACTIVE = `const INTERACTIVE_TAGS = new Set(['input', 'textarea', 'code', 'diff', 'markdown', 'virtual-list', 'anchored']);`;
const INTERACTIVE_PATCHED = `const INTERACTIVE_TAGS = new Set(['input', 'textarea', 'code', 'diff', 'markdown', 'virtual-list', 'anchored', 'polymark-editor']);`;
const TEXT_INPUTS = `const TEXT_INPUTS = new Set(['input', 'textarea']);`;
const TEXT_INPUTS_PATCHED = `const TEXT_INPUTS = new Set(['input', 'textarea', 'polymark-editor']);`;

export function patchRendererSource(source) {
    if (source.includes(MARKER)) {
        return { source, changed: false };
    }
    const missing = [];
    if (!source.includes(MAP_TAG)) missing.push('map_tag');
    if (!source.includes(INTERACTIVE)) missing.push('INTERACTIVE_TAGS');
    if (!source.includes(TEXT_INPUTS)) missing.push('TEXT_INPUTS');
    if (missing.length > 0) {
        throw new Error(
            `gpuix-svelte renderer does not match the pinned 0.5.0 patch points: ${missing.join(', ')}`
        );
    }
    let next = source.replace(MAP_TAG, MAP_TAG_PATCHED);
    next = next.replace(INTERACTIVE, INTERACTIVE_PATCHED);
    next = next.replace(TEXT_INPUTS, TEXT_INPUTS_PATCHED);
    return { source: next, changed: true };
}

function rendererPath() {
    const here = dirname(fileURLToPath(import.meta.url));
    return join(here, '..', 'node_modules', 'gpuix-svelte', 'lib', 'renderer.js');
}

function main() {
    const path = rendererPath();
    const current = readFileSync(path, 'utf8');
    const { source, changed } = patchRendererSource(current);
    if (changed) writeFileSync(path, source);
    console.log(changed ? `patched ${path}` : `already patched ${path}`);
}

const invoked = process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1];
if (invoked) main();
