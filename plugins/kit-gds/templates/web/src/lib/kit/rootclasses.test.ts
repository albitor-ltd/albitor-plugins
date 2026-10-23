// @vitest-environment jsdom
// GOV.UK Frontend's page template puts `js-enabled` and `govuk-frontend-supported` on <body> from
// an inline script. This kit does it from AppShell on mount (no inline script), so the classes
// must arrive once the shell is mounted and leave with it.
import { flushSync, mount, unmount } from 'svelte';
import { afterEach, describe, expect, it } from 'vitest';
import Showcase from './Showcase.svelte';

let app: ReturnType<typeof mount> | undefined;

// jsdom has no `noModule`; a module-capable browser does. Add and remove it per test.
const proto = HTMLScriptElement.prototype;
const supportModules = () => Object.defineProperty(proto, 'noModule', { value: false, configurable: true });

afterEach(() => {
	delete (proto as { noModule?: boolean }).noModule;
	if (app) unmount(app);
	app = undefined;
	document.body.className = '';
	document.body.innerHTML = '';
});

describe('GOV.UK Frontend body classes', () => {
	it('adds js-enabled and govuk-frontend-supported when the shell mounts', () => {
		supportModules();
		app = mount(Showcase, { target: document.body });
		flushSync();
		expect(document.body.classList.contains('js-enabled')).toBe(true);
		expect(document.body.classList.contains('govuk-frontend-supported')).toBe(true);
	});

	it('adds only js-enabled in a browser without module support', () => {
		app = mount(Showcase, { target: document.body });
		flushSync();
		expect(document.body.classList.contains('js-enabled')).toBe(true);
		expect(document.body.classList.contains('govuk-frontend-supported')).toBe(false);
	});

	it('keeps classes already on the body and removes only its own on unmount', () => {
		supportModules();
		document.body.className = 'govuk-template__body js-enabled';
		app = mount(Showcase, { target: document.body });
		flushSync();
		expect(document.body.classList.contains('govuk-frontend-supported')).toBe(true);
		unmount(app);
		app = undefined;
		expect(document.body.classList.contains('govuk-template__body')).toBe(true);
		expect(document.body.classList.contains('js-enabled')).toBe(true);
		expect(document.body.classList.contains('govuk-frontend-supported')).toBe(false);
	});
});
