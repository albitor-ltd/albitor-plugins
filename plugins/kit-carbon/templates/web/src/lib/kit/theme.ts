// Seeded by Albitor (kit-carbon), then yours: which Carbon theme the page wears.
//
// Carbon ships four themes. The kit uses two: `white` (light) and `g100` (dark), the pair Carbon
// itself pairs for light and dark. The page follows the visitor's colour-scheme preference unless
// <html data-theme="light"> or data-theme="dark" forces one, the same switch the other kits use.

export type CarbonTheme = 'white' | 'g100';

/** The theme for a forced scheme (the `data-theme` attribute) or, failing that, the preference. */
export function pickTheme(forced: string | null | undefined, prefersDark: boolean): CarbonTheme {
	if (forced === 'light') return 'white';
	if (forced === 'dark') return 'g100';
	return prefersDark ? 'g100' : 'white';
}
