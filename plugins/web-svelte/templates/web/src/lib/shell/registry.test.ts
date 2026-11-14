import { describe, expect, it, vi } from 'vitest';
import { collectRegistry } from './registry';

interface Entry {
	order: number;
	label: string;
}

function parse(value: unknown): Entry | string {
	const v = value as Partial<Entry> | null | undefined;
	if (!v || typeof v.label !== 'string') return 'label must be a string';
	return { order: v.order as number, label: v.label };
}

const mod = (entry: unknown) => ({ item: entry });

describe('collectRegistry', () => {
	it('orders by order then module path and records where each entry came from', () => {
		const got = collectRegistry<Entry>(
			{
				'/src/features/b/item.ts': mod({ order: 10, label: 'B' }),
				'/src/features/a/item.ts': mod({ order: 10, label: 'A' }),
				'/src/features/c/item.ts': mod({ order: 1, label: 'C' })
			},
			'item',
			parse,
			true
		);
		expect(got.map((e) => e.label)).toEqual(['C', 'A', 'B']);
		expect(got[0]!.source).toBe('/src/features/c/item.ts');
	});

	it('treats a parse that throws like a returned reason', () => {
		const boom = () => {
			throw new Error('boom');
		};
		const modules = { '/src/x/item.ts': mod({ order: 1, label: 'X' }) };
		expect(() => collectRegistry(modules, 'item', boom, true)).toThrow(/boom/);
		const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
		expect(collectRegistry(modules, 'item', boom, false)).toEqual([]);
		expect(warn).toHaveBeenCalledOnce();
		warn.mockRestore();
	});

	it('is empty with no modules', () => {
		expect(collectRegistry({}, 'item', parse, true)).toEqual([]);
	});

	it('throws on an unusable entry in development, naming the file', () => {
		expect(() =>
			collectRegistry({ '/src/x/item.ts': mod({ order: 1 }) }, 'item', parse, true)
		).toThrow(/\/src\/x\/item\.ts/);
		expect(() => collectRegistry({ '/src/x/item.ts': {} }, 'item', parse, true)).toThrow();
		expect(() =>
			collectRegistry({ '/src/x/item.ts': mod({ order: 5000, label: 'X' }) }, 'item', parse, true)
		).toThrow(/order/);
	});

	it('skips an unusable entry in production and keeps the rest', () => {
		const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
		const got = collectRegistry<Entry>(
			{
				'/src/bad/item.ts': mod({ order: 1.5, label: 'Bad' }),
				'/src/good/item.ts': mod({ order: 1, label: 'Good' })
			},
			'item',
			parse,
			false
		);
		expect(got.map((e) => e.label)).toEqual(['Good']);
		expect(warn).toHaveBeenCalledOnce();
		warn.mockRestore();
	});
});
