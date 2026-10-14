import { describe, expect, it } from 'vitest';
import { mergeScope } from './scope';

describe('mergeScope', () => {
	it('merges distinct exports', () => {
		expect(mergeScope({ './a.ts': { A: 1 }, './b.ts': { B: 2 } })).toEqual({ A: 1, B: 2 });
	});

	it('throws on a duplicate export name', () => {
		expect(() => mergeScope({ './a.ts': { A: 1 }, './b.ts': { A: 2 } })).toThrow(
			/"A".*a\.ts.*b\.ts/
		);
	});
});
