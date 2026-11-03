import { afterEach, describe, expect, it, vi } from 'vitest';
import { checkFile, deleteFile, downloadLink, uploadFile } from './upload';

const json = (status: number, body: unknown) =>
	new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

const limits = { maxSize: 1024 * 1024, types: ['image/png'] };

afterEach(() => vi.unstubAllGlobals());

describe('checkFile', () => {
	it('refuses an empty file, a big one and a type that is not accepted', () => {
		expect(checkFile(new File([], 'a.png', { type: 'image/png' }), limits)).toMatch(/empty/);
		expect(checkFile(new File([new Uint8Array(2 * 1024 * 1024)], 'a.png', { type: 'image/png' }), limits)).toMatch(/larger/);
		expect(checkFile(new File(['x'], 'a.html', { type: 'text/html' }), limits)).toMatch(/not accepted/);
		expect(checkFile(new File(['x'], 'a.png', { type: 'image/png' }), limits)).toBeNull();
	});
});

describe('uploadFile', () => {
	it('asks the API, PUTs the bytes to the signed URL with its headers, then completes', async () => {
		const calls: Array<[string, RequestInit]> = [];
		vi.stubGlobal(
			'fetch',
			vi.fn(async (url: string, init: RequestInit) => {
				calls.push([url, init]);
				if (url === '/api/records/r1/files') {
					return json(201, {
						file_id: 'f1',
						name: 'a.png',
						upload: { method: 'PUT', url: 'https://bucket.example/o/r1/f1?sig', headers: { 'Content-Type': 'image/png' }, expires_at: '' }
					});
				}
				if (url.startsWith('https://bucket.example')) return new Response(null, { status: 200 });
				return json(200, { id: 'f1', name: 'a.png', content_type: 'image/png', size: 1 });
			})
		);
		const file = new File(['x'], 'a.png', { type: 'image/png' });
		const out = await uploadFile('r1', file);
		expect(out).toMatchObject({ ok: true, file: { id: 'f1' } });
		expect(JSON.parse(calls[0]![1].body as string)).toEqual({ name: 'a.png', content_type: 'image/png', size: 1 });
		expect(calls[1]![0]).toBe('https://bucket.example/o/r1/f1?sig');
		expect(calls[1]![1]).toMatchObject({ method: 'PUT', headers: { 'Content-Type': 'image/png' }, body: file });
		expect(calls[2]![0]).toBe('/api/records/r1/files/f1/complete');
	});

	it('shows the API refusal and never touches the bucket', async () => {
		const fetch = vi.fn(async () => json(422, { error: 'the file is larger than 10 bytes' }));
		vi.stubGlobal('fetch', fetch);
		const out = await uploadFile('r1', new File(['x'], 'a.png', { type: 'image/png' }));
		expect(out).toEqual({ ok: false, code: '422', message: 'the file is larger than 10 bytes' });
		expect(fetch).toHaveBeenCalledTimes(1);
	});

	it('does not complete when the bucket refuses the PUT', async () => {
		const fetch = vi.fn(async (url: string) =>
			url.startsWith('/api/')
				? json(201, { file_id: 'f1', name: 'a', upload: { method: 'PUT', url: 'https://bucket.example/k', expires_at: '' } })
				: new Response(null, { status: 403 })
		);
		vi.stubGlobal('fetch', fetch);
		const out = await uploadFile('r1', new File(['x'], 'a.png', { type: 'image/png' }));
		expect(out.ok).toBe(false);
		expect(fetch).toHaveBeenCalledTimes(2);
	});
});

describe('links and delete', () => {
	it('encodes the ids in a download link', () => {
		expect(downloadLink('a/b', 'f 1')).toBe('/api/records/a%2Fb/files/f%201?redirect=1');
	});

	it('deletes through the API', async () => {
		const fetch = vi.fn(async () => new Response(null, { status: 204 }));
		vi.stubGlobal('fetch', fetch);
		expect(await deleteFile('r1', 'f1')).toEqual({ ok: true });
		expect((fetch.mock.calls[0] as unknown as [string, RequestInit])[1].method).toBe('DELETE');
	});
});
