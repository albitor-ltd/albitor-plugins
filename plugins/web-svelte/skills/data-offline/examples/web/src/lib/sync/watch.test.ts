import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import { SYNC_KEY } from './context';
import { createSyncEngine } from './engine';
import { memoryStores } from './memory';
import SyncStatus from './SyncStatus.svelte';
import { FakeServer } from './testing';
import { watchEngine, watchOnline } from './watch';

const tick = () => new Promise((r) => setTimeout(r, 0));

async function engine() {
	const e = createSyncEngine({ stores: memoryStores(), transport: new FakeServer(), collections: ['todos'], lock: (_n, fn) => fn() });
	await e.attach('alice');
	return e;
}

describe('watchEngine', () => {
	it('reads at once and again after every change, until stopped', async () => {
		const e = await engine();
		const seen: number[] = [];
		const stop = watchEngine(e, async () => (await e.list('todos')).length, (n) => seen.push(n));
		await tick();
		await e.create('todos', { title: 'a' });
		await tick();
		expect(seen.at(0)).toBe(0);
		expect(seen.at(-1)).toBe(1);
		stop();
		const count = seen.length;
		await e.create('todos', { title: 'b' });
		await tick();
		expect(seen.length).toBe(count);
	});

	it('drops a slow read that a newer one has overtaken', async () => {
		const e = await engine();
		const seen: string[] = [];
		const slow: Array<() => void> = [];
		let n = 0;
		const read = () => {
			const which = `read${++n}`;
			return new Promise<string>((resolve) => slow.push(() => resolve(which)));
		};
		const stop = watchEngine(e, read, (v) => seen.push(v));
		await e.create('todos', { title: 'a' }); // a second read starts
		slow[1]();
		await tick();
		slow[0](); // the first read finishes last, with stale data
		await tick();
		expect(seen).toEqual(['read2']);
		stop();
	});

	it('survives a read that fails because the person detached', async () => {
		const e = await engine();
		const seen: unknown[] = [];
		const stop = watchEngine(e, () => e.list('todos'), (v) => seen.push(v));
		await tick();
		e.detach();
		await tick();
		expect(seen).toHaveLength(1);
		stop();
	});
});

describe('watchOnline', () => {
	it('reports the starting state and each change', () => {
		const target = new EventTarget();
		const seen: boolean[] = [];
		const stop = watchOnline(target, true, (v) => seen.push(v));
		target.dispatchEvent(new Event('offline'));
		target.dispatchEvent(new Event('online'));
		stop();
		target.dispatchEvent(new Event('offline'));
		expect(seen).toEqual([true, false, true]);
	});
});

describe('SyncStatus', () => {
	it('renders a hidden, empty status line before anyone is attached (server render)', () => {
		const e = createSyncEngine({ stores: memoryStores(), transport: new FakeServer(), collections: ['todos'] });
		const html = render(SyncStatus, { context: new Map([[SYNC_KEY, e]]) }).body;
		expect(html).toContain('role="status"');
		expect(html).toContain('hidden');
	});
});
