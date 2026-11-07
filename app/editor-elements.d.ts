/**
 * Augments the gpuix-svelte 0.5.0 element namespace.
 * `polymark-editor` is a hyphenated custom element. The published renderer
 * maps it to `div` until scripts/patch-hyphenated-tags.mjs runs.
 */
declare global {
	namespace gpuixHTML {
		interface IntrinsicElements {
			'polymark-editor': IntrinsicElements['div'] & {
				source?: string | null;
			};
		}
	}
}

export {};
