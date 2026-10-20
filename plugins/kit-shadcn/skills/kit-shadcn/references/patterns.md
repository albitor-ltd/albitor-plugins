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
choice (Close) always available. Name the thing being deleted in the title.

## A list page

`Table` for the rows; the same page shows `EmptyState` when there are none, with an `action`
snippet that starts the first one. Fetch in `+page.ts` with the API client and handle
`NetworkError` (check `.offline`) and `ApiError` with a visible message.

## Errors and loading

Show a loading state in the place the content will appear, and an error with a retry `Button`
there too, not only a toast: a toast is gone once dismissed.

## Navigation on a phone

Every desktop nav destination stays reachable at phone width; never hide a control. Pick one
pattern per app:

- 2 to 5 top-level destinations people switch between often: a bottom tab bar.
- More than 5, or infrequent or hierarchical navigation: a menu button that opens a drawer, or,
  where the look prefers a tab bar, 4 tabs plus a "More" tab that opens the remaining destinations
  as a list.
- Never a tab bar and a menu button carrying the same destinations.

The kit has no tab bar or drawer component; build them from the existing pieces and tokens, beside
`AppShell`. `AppShell`'s header nav is hidden below 48rem (`display: none`) and the phone nav
replaces it, so only one nav is exposed at a time.

**Tab bar.** A `<nav aria-label="Primary">` fixed to the bottom, `display: flex`, each tab a link
with an icon and a visible text label (never icon-only). Put `aria-current="page"` on the current
tab and style it from `[aria-current]` (as `AppShell` does, which also marks parent sections with
`aria-current="true"`), using weight or an indicator bar plus `var(--primary)`, not colour alone.
Tabs are at least 44x44 CSS px (WCAG 2.2 2.5.8 requires 24x24; 44 is the house recommendation).
Background `var(--background)` with a `var(--border)` top line, and
`padding-bottom: env(safe-area-inset-bottom)` so it clears the home indicator (`viewport-fit=cover`
is set in web-svelte's `app.html`). Give `main` matching bottom padding so content is never hidden.

**More than 5 on a tab bar.** Show 4 tabs plus a fifth "More" tab, a real `<button>` that opens the
rest as a list in the drawer below. The current destination stays indicated even when it lives under
More, but never put `aria-current="page"` on the More button: keep it on the item inside the opened
list. Give More a visual indicator plus visually hidden text such as "(current page is in More)".

**Menu button and drawer.** The drawer is a modal dialog containing a `<nav>`, not a disclosure. The
kit's `Dialog.svelte` is centred, needs title and description props, and portals its content only
when open, so do not hand-roll `aria-controls` against it. Use the Bits UI `Dialog` primitives
directly for a left- or right-anchored panel:

- `Dialog.Trigger` is the menu button: a real `<button>` with an accessible name ("Menu"); Bits wires
  `aria-expanded` and `aria-controls` for you.
- `Dialog.Content` is the anchored panel, styled with kit tokens. Give it a `Dialog.Title` (visually
  hidden is fine) as its accessible name.
- Bits moves focus into the panel on open, closes on Escape and returns focus to the button, traps
  focus and makes the background inert. Close it when a link or the scrim is activated.
- Inside, a labelled `<nav>` with `aria-current="page"` on the current link; links at least 44px tall.

Test at 390px wide: every desktop destination is reachable, nothing overflows horizontally, and
keyboard and screen reader users can open, use and close the navigation.
