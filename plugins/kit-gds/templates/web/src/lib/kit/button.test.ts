// Button's class handling, rendered on the server on its own. Delete this file with Button.svelte
// if the app has no button.
import { describe, expect, it } from 'vitest';
import { createRawSnippet } from 'svelte';
import { render } from 'svelte/server';
import Button from './Button.svelte';

describe('Button', () => {
	const children = createRawSnippet(() => ({ render: () => '<span>Go</span>' }));

	it('merges a caller\'s class with govuk-button instead of replacing it', () => {
		const { body } = render(Button, { props: { class: 'my-extra', variant: 'destructive', children } });
		expect(body).toMatch(/class="govuk-button govuk-button--warning my-extra"/);
	});

	it('keeps govuk-button with no caller class', () => {
		const { body } = render(Button, { props: { children } });
		expect(body).toMatch(/class="govuk-button"/);
	});
});
