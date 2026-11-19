import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { isSafeRedirect, runGuards } from '../shell/guard';
import { AUTH_PAGES, PUBLIC_PATHS, isPublicPath, nextPath, requireSession, signInPath } from './guard';

const session = vi.hoisted(() => ({ getSession: vi.fn() }));
vi.mock('./session', () => session);

const at = (path: string) => new URL(path, 'https://app.test');

beforeEach(() => {
	session.getSession.mockReset();
});

describe('requireSession', () => {
	it('sends a signed-out person to sign-in, remembering where they were headed', async () => {
		session.getSession.mockResolvedValue(null);
		expect(await requireSession({ url: at('/reports?tab=2') })).toBe(
			'/sign-in?next=%2Freports%3Ftab%3D2'
		);
	});

	it('lets a signed-in person through', async () => {
		session.getSession.mockResolvedValue({ sub: 'u1' });
		expect(await requireSession({ url: at('/reports') })).toBeUndefined();
	});

	it.each(AUTH_PAGES)('lets anyone reach %s, without asking the API', async (path) => {
		expect(await requireSession({ url: at(path) })).toBeUndefined();
		expect(session.getSession).not.toHaveBeenCalled();
	});

	it('does not read a failed session check as signed out', async () => {
		session.getSession.mockRejectedValue(new Error('API down'));
		await expect(requireSession({ url: at('/reports') })).rejects.toThrow('API down');
	});

	it('only ever redirects to somewhere the shell accepts, and to somewhere it allows', async () => {
		session.getSession.mockResolvedValue(null);
		for (const path of ['/', '/a/b', '/a?x=%2F%2Fevil.test', '/%5Cevil.test']) {
			const to = (await requireSession({ url: at(path) }))!;
			expect(isSafeRedirect(to)).toBe(true);
			expect(isPublicPath(new URL(to, 'https://app.test').pathname)).toBe(true);
		}
		// The shell's own runner agrees: it returns the redirect rather than throwing.
		expect(await runGuards([requireSession], at('/reports'))).toBe('/sign-in?next=%2Freports');
	});
});

describe('signInPath', () => {
	it('keeps the whole target in one encoded parameter, so it cannot add another', () => {
		const to = signInPath(at('/x?a=1&next=//evil.test'));
		expect(new URL(to, 'https://app.test').searchParams.get('next')).toBe('/x?a=1&next=//evil.test');
		expect(new URL(to, 'https://app.test').searchParams.getAll('next')).toHaveLength(1);
	});
});

describe('nextPath', () => {
	it.each([
		['/reports', '/reports'],
		['/reports?tab=2', '/reports?tab=2'],
		['/a/b#top', '/a/b#top']
	])('follows %s', (next, want) => {
		expect(nextPath(next)).toBe(want);
	});

	it.each([
		null,
		undefined,
		'',
		'//evil.test',
		'/\\evil.test',
		'https://evil.test/',
		'javascript:alert(1)',
		'reports',
		'/sign-in',
		'/sign-up?x=1'
	])('falls back to / for %s', (next) => {
		expect(nextPath(next)).toBe('/');
	});
});

describe('declared public pages (browse posture)', () => {
	// The foundation node adds the manifest's web entries to PUBLIC_PATHS; the test does the same.
	const declared = ['/listings', '/listings/*'];
	beforeEach(() => {
		(PUBLIC_PATHS as string[]).push(...declared);
	});
	afterEach(() => {
		(PUBLIC_PATHS as string[]).splice(-declared.length, declared.length);
	});

	it('matches an exact entry, and a /* entry only below its prefix', () => {
		expect(isPublicPath('/listings')).toBe(true);
		expect(isPublicPath('/listings/42')).toBe(true);
		expect(isPublicPath('/listings/42/photos')).toBe(true);
		expect(isPublicPath('/listings/')).toBe(false);
		expect(isPublicPath('/listingsx')).toBe(false);
		expect(isPublicPath('/saved')).toBe(false);
		expect(isPublicPath('/other/listings/1')).toBe(false);
	});

	it('lets a signed-out visitor open them without asking the API, and still guards the rest', async () => {
		expect(await requireSession({ url: at('/listings/42') })).toBeUndefined();
		expect(session.getSession).not.toHaveBeenCalled();
		session.getSession.mockResolvedValue(null);
		expect(await requireSession({ url: at('/saved?x=1') })).toBe('/sign-in?next=%2Fsaved%3Fx%3D1');
	});

	it('returns a person to a public page they signed in from, but never to an auth page', () => {
		expect(nextPath('/listings/42?q=a')).toBe('/listings/42?q=a');
		expect(nextPath('/sign-in')).toBe('/');
	});
});
