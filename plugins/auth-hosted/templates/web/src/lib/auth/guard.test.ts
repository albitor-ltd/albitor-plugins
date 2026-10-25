import { beforeEach, describe, expect, it, vi } from 'vitest';
import { isSafeRedirect, runGuards } from '../shell/guard';
import { PUBLIC_PATHS, isPublicPath, nextPath, requireSession, signInPath } from './guard';

const session = vi.hoisted(() => ({ getSession: vi.fn(), takeRefusal: vi.fn(() => false) }));
vi.mock('./session', () => session);

const at = (path: string) => new URL(path, 'https://app.test');

beforeEach(() => {
	session.getSession.mockReset();
	session.takeRefusal.mockReset();
	session.takeRefusal.mockReturnValue(false);
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

	it.each(PUBLIC_PATHS)('lets anyone reach %s, without asking the API', async (path) => {
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

describe('a refresh the API refused', () => {
	it('sends the person to the refusal page, not round sign-in', async () => {
		session.getSession.mockResolvedValue(null);
		session.takeRefusal.mockReturnValue(true);
		expect(await requireSession({ url: at('/reports') })).toBe('/not-allowed');
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
		'/sign-in?x=1',
		'/sign-out',
		'/not-allowed'
	])('falls back to / for %s', (next) => {
		expect(nextPath(next)).toBe('/');
	});
});
