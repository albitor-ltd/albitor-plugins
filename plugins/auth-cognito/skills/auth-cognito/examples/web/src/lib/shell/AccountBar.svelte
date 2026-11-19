<!-- Replaces the shell's empty AccountBar. Signed in it holds Sign out; signed out (the nav is
     shown there only on a declared public page, browse posture) it holds Sign in and Sign up, which carry
     where the visitor was as `next`. Sign up is the app's /sign-up screen; an invite-only app sends it on to /sign-in.
     Sign out is a POST form button, never a link: a link can be prefetched, or followed from a
     page the visitor does not control, and either would sign them out. The browser posts the form
     and the API answers 303 to /sign-in with the session cookies cleared. The API refuses a form
     post that did not come from this site. Work to finish first (wiping the data layer's local
     copy) is registered with onBeforeSignOut in lib/shell/signout.ts; without script the form
     still posts. -->
<script lang="ts">
	import { page } from '$app/state';
	import { runBeforeSignOut } from './signout';

	const here = $derived(encodeURIComponent(page.url.pathname + page.url.search));
	let busy = $state(false);
	let failed = $state(false);

	async function onsubmit(event: SubmitEvent) {
		const form = event.currentTarget as HTMLFormElement;
		event.preventDefault();
		busy = true;
		failed = false;
		if (await runBeforeSignOut()) {
			form.submit();
			return;
		}
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
		<a href="/sign-up?next={here}" data-testid="sign-up-link">Sign up</a>
	</nav>
{/if}
