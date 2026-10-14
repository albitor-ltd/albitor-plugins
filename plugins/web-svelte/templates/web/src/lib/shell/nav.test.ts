import { describe, expect, it, vi } from 'vitest';
import { buildNav, hrefForLabel, hrefForNavFile, parseNavMeta } from './nav';

describe('hrefForNavFile', () => {
	it('maps a route directory to its link and drops route groups', () => {
		expect(hrefForNavFile('/src/routes/nav.ts')).toBe('/');
		expect(hrefForNavFile('/src/routes/reports/nav.ts')).toBe('/reports');
		expect(hrefForNavFile('/src/routes/(app)/reports/monthly/nav.ts')).toBe('/reports/monthly');
	});

	it('refuses parameter routes and files outside the routes tree', () => {
		expect(hrefForNavFile('/src/routes/items/[id]/nav.ts')).toBeNull();
		expect(hrefForNavFile('/src/routes/items/[[id]]/nav.ts')).toBeNull();
		expect(hrefForNavFile('/src/lib/nav.ts')).toBeNull();
		expect(hrefForNavFile('/src/routes/reports/other.ts')).toBeNull();
	});
});

describe('hrefForLabel', () => {
	it('slugs a label into a link', () => {
		expect(hrefForLabel('Sign in')).toBe('/sign-in');
		expect(hrefForLabel('Reports & exports (beta)')).toBe('/reports-exports-beta');
	});
});

describe('parseNavMeta', () => {
	it('accepts the documented shape', () => {
		expect(parseNavMeta({ label: 'Home', order: 0 })).toEqual({ label: 'Home', order: 0 });
		expect(parseNavMeta({ label: 'Home', order: 10, group: 'main' })).toEqual({
			label: 'Home',
			order: 10,
			group: 'main'
		});
	});

	it.each([
		[null],
		['Home'],
		[{ label: '', order: 1 }],
		[{ label: 'x'.repeat(65), order: 1 }],
		[{ label: '<b>Home</b>', order: 1 }],
		[{ label: 'Home', order: 1.5 }],
		[{ label: 'Home', order: 1001 }],
		[{ label: 'Home', order: -1 }],
		[{ label: 'Home', order: '1' }],
		[{ label: 'Home', order: 1, group: 'Main' }],
		[{ label: 'Home', order: 1, href: '/x' }]
	])('rejects %j', (value) => {
		expect(typeof parseNavMeta(value)).toBe('string');
	});
});

describe('buildNav', () => {
	const route = (nav: unknown) => ({ nav });

	it('orders by order then label, putting ungrouped entries first and groups by id', () => {
		const groups = buildNav(
			[{ label: 'Account', order: 5, group: 'user' }],
			{
				'/src/routes/nav.ts': route({ label: 'Home', order: 0 }),
				'/src/routes/(app)/b/nav.ts': route({ label: 'Bravo', order: 20, group: 'data' }),
				'/src/routes/(app)/a/nav.ts': route({ label: 'Alpha', order: 20, group: 'data' }),
				'/src/routes/(app)/c/nav.ts': route({ label: 'Charlie', order: 10, group: 'data' })
			},
			true
		);
		expect(groups.map((g) => g.id)).toEqual([null, 'data', 'user']);
		expect(groups[1]!.items.map((i) => i.label)).toEqual(['Charlie', 'Alpha', 'Bravo']);
		expect(groups[1]!.items.map((i) => i.href)).toEqual(['/c', '/a', '/b']);
		expect(groups[2]!.items[0]!.href).toBe('/account');
	});

	it('keeps the route entry when a plugin names the same link', () => {
		const groups = buildNav(
			[{ label: 'Account', order: 5 }],
			{ '/src/routes/account/nav.ts': route({ label: 'Account', order: 50 }) },
			true
		);
		expect(groups).toHaveLength(1);
		expect(groups[0]!.items).toEqual([{ label: 'Account', order: 50, href: '/account' }]);
	});

	it('throws on a bad entry in development', () => {
		expect(() =>
			buildNav([], { '/src/routes/x/nav.ts': route({ label: 'X', order: 5000 }) }, true)
		).toThrow(/\/src\/routes\/x\/nav\.ts/);
		expect(() => buildNav([], { '/src/routes/x/nav.ts': {} }, true)).toThrow();
		expect(() =>
			buildNav([], { '/src/routes/[id]/nav.ts': route({ label: 'X', order: 1 }) }, true)
		).toThrow();
	});

	it('skips a bad entry in production and keeps the rest', () => {
		const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
		const groups = buildNav(
			[],
			{
				'/src/routes/bad/nav.ts': route({ label: 'Bad', order: -3 }),
				'/src/routes/good/nav.ts': route({ label: 'Good', order: 1 })
			},
			false
		);
		expect(groups.flatMap((g) => g.items.map((i) => i.label))).toEqual(['Good']);
		expect(warn).toHaveBeenCalledOnce();
		warn.mockRestore();
	});

	it('refuses a symbol-only plugin label', () => {
		expect(() => hrefForLabel('***')).toThrow(RangeError);
		expect(() => buildNav([{ label: '&&', order: 1 }], {}, true)).toThrow(/no letters or digits/);
	});

	it('treats two plugin entries with one link as an error', () => {
		const dup = [
			{ label: 'Sign in', order: 1 },
			{ label: 'Sign-in', order: 2 }
		];
		expect(() => buildNav(dup, {}, true)).toThrow(/duplicate link \/sign-in/);
	});

	it('returns no groups for an empty app', () => {
		expect(buildNav([], {}, true)).toEqual([]);
	});
});
