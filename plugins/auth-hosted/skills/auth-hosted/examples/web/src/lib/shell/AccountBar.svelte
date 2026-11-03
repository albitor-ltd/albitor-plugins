<!-- Replaces the shell's empty AccountBar. Shown only to someone signed in.
     Sign out is a POST form button, never a link: a link can be prefetched, or followed from a
     page the visitor does not control, and either would sign them out. The browser posts the form
     and the API answers 303 to Cognito's sign-out page (which clears Cognito's own cookie, so the
     next "Sign in" asks for the password again) with the session cookies cleared. The API refuses a form
     post that did not come from this site. Work to finish first (wiping the data layer's local
     copy) is registered with onBeforeSignOut in lib/shell/signout.ts; without script the form
     still posts. -->
<script lang="ts">
	import { runBeforeSignOut } from './signout';

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

<form method="POST" action="/api/auth/sign-out" data-testid="sign-out-form" {onsubmit}>
	<button type="submit" data-testid="sign-out-button" disabled={busy}>Sign out</button>
	{#if failed}
		<p role="alert" data-testid="sign-out-failed">You were not signed out: some changes have not been saved.</p>
	{/if}
</form>
