import { describe, expect, it, vi } from 'vitest';
import { onBeforeSignOut, runBeforeSignOut } from './signout';

describe('runBeforeSignOut', () => {
	it('is true with no hooks, and awaits each hook in order', async () => {
		expect(await runBeforeSignOut()).toBe(true);
		const order: string[] = [];
		const off1 = onBeforeSignOut(async () => {
			await Promise.resolve();
			order.push('a');
		});
		const off2 = onBeforeSignOut(() => void order.push('b'));
		expect(await runBeforeSignOut()).toBe(true);
		expect(order).toEqual(['a', 'b']);
		off1();
		off2();
	});

	it('stops when a hook returns false or throws, and a removed hook no longer runs', async () => {
		const off = onBeforeSignOut(() => false);
		expect(await runBeforeSignOut()).toBe(false);
		off();
		const off2 = onBeforeSignOut(() => {
			throw new Error('disk');
		});
		expect(await runBeforeSignOut()).toBe(false);
		off2();
		const never = vi.fn();
		expect(await runBeforeSignOut()).toBe(true);
		expect(never).not.toHaveBeenCalled();
	});
});
