import assert from 'node:assert/strict';
import test from 'node:test';
import { patchRendererSource } from './patch-hyphenated-tags.mjs';

const SAMPLE = `const INTERACTIVE_TAGS = new Set(['input', 'textarea', 'code', 'diff', 'markdown', 'virtual-list', 'anchored']);
const TEXT_INPUTS = new Set(['input', 'textarea']);
function map_tag(name) {
    if (GPUI_TAGS.has(name))
        return name;
    if (!warned_tags.has(name)) {
        warned_tags.add(name);
        console.warn(\`[gpuix-svelte] <\${name}> has no GPUI equivalent — rendering it as <div>.\`);
    }
    return 'div';
}
`;

test('hyphenated tags pass through and div fallback stays', () => {
    const { source, changed } = patchRendererSource(SAMPLE);
    assert.equal(changed, true);
    assert.match(source, /name\.includes\('-'\)/);
    assert.match(source, /return 'div'/);
    assert.match(source, /'polymark-editor'/);
    const again = patchRendererSource(source);
    assert.equal(again.changed, false);
    assert.equal(again.source, source);
});

test('a drifted renderer is refused', () => {
    assert.throws(() => patchRendererSource('function map_tag() { return 1; }\n'), /pinned 0\.5\.0/);
});
