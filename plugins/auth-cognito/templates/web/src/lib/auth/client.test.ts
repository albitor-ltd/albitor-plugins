import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { confirm, forgot, resend, reset, signIn, signOut, signUp } from './client';

const session = vi.hoisted(() => ({ clearSession: vi.fn() }));
vi.mock('./session', () => session);

const json = (status: number, body: unknown) =>
	new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

function stubFetch(impl: (url: string, init: RequestInit) => Response | Promise<Response>) {
	const fn = vi.fn(async (url: string, init: RequestInit) => impl(url, init));
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => session.clearSession.mockReset());
afterEach(() => vi.unstubAllGlobals());

describe('the sign-in calls', () => {
	it('post JSON to /api/auth/<action>', async () => {
		const fetch = stubFetch(() => json(204, undefined));
		await signIn('a@example.test', 'pw');
		await confirm('a@example.test', '123456');
		await resend('a@example.test');
		await forgot('a@example.test');
		await reset('a@example.test', '123456', 'new-pw');
		const sent = fetch.mock.calls.map(([url, init]) => [url, init.method, init.body]);
		expect(sent).toEqual([
			['/api/auth/sign-in', 'POST', '{"email":"a@example.test","password":"pw"}'],
			['/api/auth/confirm', 'POST', '{"email":"a@example.test","code":"123456"}'],
			['/api/auth/resend', 'POST', '{"email":"a@example.test"}'],
			['/api/auth/forgot', 'POST', '{"email":"a@example.test"}'],
			['/api/auth/reset', 'POST', '{"email":"a@example.test","code":"123456","password":"new-pw"}']
		]);
	});

	it('answer ok, with the body where there is one', async () => {
		stubFetch(() => json(201, { confirmation_required: true }));
		expect(await signUp('a@example.test', 'pw')).toEqual({ ok: true, confirmation_required: true });
		stubFetch(() => new Response(null, { status: 204 }));
		expect(await confirm('a@example.test', '1')).toEqual({ ok: true });
	});

	it('answer a refusal as a result carrying the API code, and do not throw', async () => {
		stubFetch(() =>
			json(401, { error: 'incorrect email or password', code: 'invalid_credentials' })
		);
		expect(await signIn('a@example.test', 'wrong')).toEqual({
			ok: false,
			code: 'invalid_credentials',
			message: 'incorrect email or password'
		});
	});

	it('answer a server fault or an unreadable body as unavailable', async () => {
		stubFetch(() => json(502, { error: 'cognito said something private', code: 'unavailable' }));
		expect(await signIn('a@example.test', 'pw')).toMatchObject({ ok: false, code: 'unavailable' });
		stubFetch(() => new Response('<html>', { status: 400 }));
		expect(await signIn('a@example.test', 'pw')).toMatchObject({ ok: false, code: 'unavailable' });
	});

	it('answer offline and unreachable apart', async () => {
		stubFetch(() => {
			throw new TypeError('failed to fetch');
		});
		vi.stubGlobal('navigator', { onLine: false });
		expect(await signIn('a@example.test', 'pw')).toMatchObject({ ok: false, code: 'offline' });
		vi.stubGlobal('navigator', { onLine: true });
		expect(await signIn('a@example.test', 'pw')).toMatchObject({ ok: false, code: 'unavailable' });
	});
});

describe('the session cache', () => {
	it('is cleared by a sign-in that worked, and not by one that failed', async () => {
		stubFetch(() => json(401, { error: 'no', code: 'invalid_credentials' }));
		await signIn('a@example.test', 'wrong');
		expect(session.clearSession).not.toHaveBeenCalled();

		stubFetch(() => json(200, { sub: 'u1' }));
		await signIn('a@example.test', 'right');
		expect(session.clearSession).toHaveBeenCalledOnce();
	});

	it('is cleared by sign-out, whatever the API said', async () => {
		stubFetch(() => json(502, {}));
		await signOut();
		expect(session.clearSession).toHaveBeenCalledOnce();
	});
});
