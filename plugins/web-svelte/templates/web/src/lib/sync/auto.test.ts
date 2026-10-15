import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { startAutoSync } from './auto';
import { createSyncEngine } from './engine';
import { memoryStores } from './memory';
import { FakeServer } from './testing';

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

async function setup() {
	const server = new FakeServer();
	const engine = createSyncEngine({ stores: memoryStores(), transport: server, collections: ['todos'], lock: (_n, fn) => fn() });
	await engine.attach('alice');
	const events = new EventTarget();
	const stop = startAutoSync(engine, { events, intervalMs: 10_000, debounceMs: 100 });
	return { server, engine, events, stop };
}

describe('startAutoSync', () => {
	it('uploads a burst of writes in one run shortly after the last', async () => {
		const { server, engine, stop } = await setup();
		await engine.create('todos', { title: 'a' });
		await engine.create('todos', { title: 'b' });
		expect(server.pushes).toHaveLength(0);
		await vi.advanceTimersByTimeAsync(150);
		expect(server.pushes).toHaveLength(2);
		stop();
	});

	it('syncs when the browser comes back online', async () => {
		const { server, engine, events, stop } = await setup();
		server.offline = true;
		await engine.create('todos', { title: 'a' });
		await vi.advanceTimersByTimeAsync(150);
		expect(server.pushes).toHaveLength(0);
		server.offline = false;
		events.dispatchEvent(new Event('online'));
		await vi.advanceTimersByTimeAsync(0);
		expect(server.pushes).toHaveLength(1);
		stop();
	});

	it('does not loop while it has nothing to do or the network is down', async () => {
		const { server, engine, stop } = await setup();
		server.offline = true;
		await engine.create('todos', { title: 'a' });
		const spy = vi.spyOn(server, 'push');
		await vi.advanceTimersByTimeAsync(5_000);
		expect(spy.mock.calls.length).toBeLessThanOrEqual(1);
		stop();
	});

	it('does nothing after it is stopped, and nothing when nobody is attached', async () => {
		const { server, engine, events, stop } = await setup();
		stop();
		await engine.create('todos', { title: 'a' });
		events.dispatchEvent(new Event('online'));
		await vi.advanceTimersByTimeAsync(60_000);
		expect(server.pushes).toHaveLength(0);

		engine.detach();
		const again = startAutoSync(engine, { events, intervalMs: 1000 });
		await vi.advanceTimersByTimeAsync(5000);
		expect(server.pushes).toHaveLength(0);
		again();
	});
});
