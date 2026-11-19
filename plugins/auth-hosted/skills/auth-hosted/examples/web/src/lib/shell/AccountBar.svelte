<!-- Replaces the shell's empty AccountBar. Signed in it holds Sign out; signed out (the nav is
     shown there only on a declared public page, browse posture) it holds Sign in and Sign up, which carry
     where the visitor was as `next`. Cognito's hosted page offers sign-up as well as sign-in, so both links go through /sign-in.
     Sign out is a POST form button, never a link: a link can be prefetched, or followed from a
     page the visitor does not control, and either would sign them out. With script (always, in a
     browser) the submit is intercepted: the work registered with onBeforeSignOut in
     lib/shell/signout.ts runs first (wiping the data layer's local copy), then the API is called by
     fetch and the page navigates to Cognito's sign-out page, which clears Cognito's own cookie (so
     the next "Sign in" asks for the password again). It is not a plain form post answered by 303:
     the hosted site's CSP has form-action 'self', which blocks a form whose redirect leaves the
     site. Without script the form still posts and the API clears the session, but the browser may
     refuse the redirect. -->
<script lang="ts">
	import { page } from '$app/state';
	import { signOutAndLeave } from '../auth/client';
	import { runBeforeSignOut } from './signout';

	const here = $derived(encodeURIComponent(page.url.pathname + page.url.search));
	let busy = $state(false);
	let failed = $state(false);

	async function onsubmit(event: SubmitEvent) {
		event.preventDefault();
		busy = true;
		failed = false;
		if ((await runBeforeSignOut()) && (await signOutAndLeave()).ok) return;
		busy = false;
		failed = true;
	}
</script>

{#if page.data.signedIn}
	<form method="POST" action="/api/auth/sign-out" data-testid="sign-out-form" {onsubmit}>
		<button type="submit" data-testid="sign-out-button" disabled={busy}>Sign out</button>
		{#if failed}
			<p role="alert" data-testid="sign-out-failed">You were not signed out: some changes have not been saved.</p>
		{/if}
	</form>
{:else}
	<nav aria-label="Account">
		<a href="/sign-in?next={here}" data-testid="sign-in-link">Sign in</a>
		<a href="/sign-in?next={here}" data-testid="sign-up-link">Sign up</a>
	</nav>
{/if}
