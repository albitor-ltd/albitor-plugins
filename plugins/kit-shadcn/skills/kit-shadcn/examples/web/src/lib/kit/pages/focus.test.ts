// @vitest-environment jsdom
// Focus after a state change, which the server-rendered page tests cannot see: ItemPage moves focus
// to the confirmation heading when the enquiry is sent, and ConfirmationPage takes focus on its h1
// when it replaces a form in place. Mounted on the client in jsdom.
import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, describe, expect, it } from 'vitest';
import ItemFixture from './__fixtures__/ItemPageFixture.svelte';
import StepFixture from './__fixtures__/StepPageFixture.svelte';

let app: ReturnType<typeof mount> | undefined;

afterEach(() => {
	if (app) unmount(app);
	app = undefined;
	document.body.innerHTML = '';
});

describe('ItemPage focus', () => {
	it('moves focus from the gone submit button to the confirmation heading when sent flips', async () => {
		app = mount(ItemFixture, { target: document.body, props: {} }) as ReturnType<typeof mount> & {
			send(): void;
		};
		flushSync();
		const submit = document.querySelector<HTMLButtonElement>('button[type="submit"]')!;
		submit.focus();
		expect(document.activeElement).toBe(submit);
		(app as unknown as { send(): void }).send();
		flushSync();
		await tick();
		expect(document.querySelector('form')).toBeNull();
		const heading = document.activeElement as HTMLElement;
		expect(heading.tagName).toMatch(/^H[2-4]$/);
		expect(heading.textContent).toBe('Thank you');
		expect(heading.getAttribute('tabindex')).toBe('-1');
	});

	it('does not move focus while the form is still showing', async () => {
		app = mount(ItemFixture, { target: document.body, props: {} });
		flushSync();
		await tick();
		expect(document.activeElement).toBe(document.body);
	});
});

describe('ConfirmationPage focus', () => {
	it('takes focus on its h1, which is focusable but not in the tab order', async () => {
		app = mount(StepFixture, { target: document.body, props: { phase: 'done' } });
		flushSync();
		await tick();
		const h1 = document.querySelector('h1')!;
		expect(h1.getAttribute('tabindex')).toBe('-1');
		expect(document.activeElement).toBe(h1);
	});
});
