# Patterns

## A form that validates

```svelte
<script lang="ts">
	import { Button, Field, toast } from '../../lib/kit';
	import { post, ApiError } from '../../lib/shell/api';

	let email = $state('');
	let error = $state('');

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		error = '';
		if (!/^\S+@\S+\.\S+$/.test(email)) {
			error = 'Enter an email address, like name@example.com';
			return;
		}
		try {
			await post('/invites', { email });
			toast.push({ message: 'Invite sent', tone: 'success' });
		} catch (e) {
			toast.push({ message: e instanceof ApiError ? 'The server refused the invite' : 'Could not reach the server', tone: 'error' });
		}
	}
</script>

<form onsubmit={submit} novalidate>
	<Field label="Email address" type="email" bind:value={email} {error} autocomplete="email" />
	<Button type="submit">Send invite</Button>
</form>
```

- Put the message next to the field (`error`), and on submit also move focus to the first invalid
  control, or summarise errors in a heading above the form.
- Say what to do, not only what is wrong. Validate on the server too: the browser is not a boundary.
- Never show a raw server error body in a toast; write a sentence.

## Confirm before a destructive action

A `Dialog` with a `destructive` button in `actions`, the consequence in `description`, and the safe
choice (the Close button) always available. Name the thing being deleted in the title.

## A list page

`Table` for the rows; the same page shows `EmptyState` when there are none, with an `action`
snippet that starts the first one. Fetch in `+page.ts` with the API client and handle
`NetworkError` (check `.offline`) and `ApiError` with a visible message. Carbon's `Grid` lays the
page out; its `Row` and `Column` take the breakpoints.

## Errors and loading

Show a loading state in the place the content will appear (Carbon's `InlineLoading` or a skeleton),
and an error with a retry `Button` there too, not only a toast: a toast is gone once dismissed.

## Navigation on a phone
`AppShell` already handles it: Carbon's UI shell closes the side nav below its 66rem breakpoint and
the header's menu button opens it as an overlay (it closes again after a navigation). The `account`
snippet (Sign out, settings) sits at the right of the header at every width. Do not add a second nav
or a hand-built tab bar carrying the same destinations. Test at 390px wide: every destination is
reachable from the menu button, nothing overflows horizontally.
