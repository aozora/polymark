/** Token names the shell reads as `var(--name)`. Values are hex colours GPUI accepts. */
export type ThemeTokens = {
	bg: string;
	fg: string;
	muted: string;
	accent: string;
	border: string;
	selection: string;
};

export type ThemeName = 'fallback' | 'paper';

/**
 * Dark writing surface used when no desktop theme is connected.
 * This is not a captured Omarchy/Omaxian palette. No theme was read from a
 * Devuan session in this environment.
 */
export const fallback_tokens: ThemeTokens = {
	bg: '#1c1b1a',
	fg: '#ece7df',
	muted: '#a39e96',
	accent: '#e6a15c',
	border: '#3a3835',
	selection: '#3d4f66'
};

/** High-contrast paper surface, to show a live token swap. */
export const paper_tokens: ThemeTokens = {
	bg: '#f4f0e6',
	fg: '#1c1b1a',
	muted: '#5e5a54',
	accent: '#8a3e16',
	border: '#d9d2c5',
	selection: '#d5e2f2'
};

export function tokens_for(name: ThemeName): ThemeTokens {
	return name === 'paper' ? paper_tokens : fallback_tokens;
}
