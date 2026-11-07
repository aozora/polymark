/** svelte-check reads this. DOM accessibility warnings do not apply to GPUI. */
export default {
	compilerOptions: {
		warningFilter: (warning) => !warning.code.startsWith('a11y_')
	}
};
