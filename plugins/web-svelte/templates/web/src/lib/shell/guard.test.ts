import { describe, expect, it } from 'vitest';
import { checkGuards, checkProviders, isSafeRedirect, runGuards } from './guard';

const url = new URL('https://app.test/reports');

describe('checkProviders and checkGuards', () => {
	it('pass resolved functions through and accept an empty slot', () => {
		const fn = () => undefined;
		expect(checkProviders([fn])).toEqual([fn]);
		expect(checkGuards([])).toEqual([]);
	});

	it('fail loudly when a declared name did not resolve', () => {
		expect(() => checkProviders([undefined])).toThrow(/web\.providers\[0\] did not resolve/);
		expect(() => checkGuards(['requireSession'])).toThrow(/web\.route-guard\[0\]/);
	});
});

describe('isSafeRedirect', () => {
	it.each(['/', '/sign-in', '/a/b?next=%2Fx', '/a#top'])('allows %s', (to) => {
		expect(isSafeRedirect(to)).toBe(true);
	});

	it.each([
		'//evil.test',
		'/\\evil.test',
		'https://evil.test/',
		'javascript:alert(1)',
		'sign-in',
		'',
		'/a\nb'
	])('refuses %j', (to) => {
		expect(isSafeRedirect(to)).toBe(false);
	});
});

describe('runGuards', () => {
	it('allows when every guard returns nothing', async () => {
		expect(await runGuards([() => undefined, async () => undefined], url)).toBeNull();
		expect(await runGuards([], url)).toBeNull();
	});

	it('returns the first redirect and stops', async () => {
		let second = false;
		const to = await runGuards(
			[
				async () => '/sign-in',
				() => {
					second = true;
				}
			],
			url
		);
		expect(to).toBe('/sign-in');
		expect(second).toBe(false);
	});

	it('hands each guard the url', async () => {
		let seen = '';
		await runGuards([({ url: u }) => void (seen = u.pathname)], url);
		expect(seen).toBe('/reports');
	});

	it('refuses an off-origin redirect', async () => {
		await expect(runGuards([() => 'https://evil.test/'], url)).rejects.toThrow(/same-origin/);
		await expect(runGuards([() => '//evil.test'], url)).rejects.toThrow(/same-origin/);
	});
});
