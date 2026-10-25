<!-- Seeded by Albitor (auth-hosted, component-kit@2), then yours: signs the person out, then goes
     through Cognito's sign-out page, which clears Cognito's own cookie and returns to the app's home
     page. Without that trip the next "Sign in" would skip the password. Its nav.ts puts "Sign out" in
     the nav; delete nav.ts to take it out. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { isHostedUrl, signOut } from '../../../lib/auth/client';
	import { Button } from '../../../lib/kit';

	let error = $state('');
	let busy = $state(true);

	async function run() {
		error = '';
		busy = true;
		const result = await signOut();
		if (result.ok && isHostedUrl(result.logout_url)) {
			window.location.assign(result.logout_url);
			return;
		}
		// A local run has no Cognito to sign out of; the session is cleared either way.
		if (result.ok || result.code === 'local') {
			await goto('/sign-in');
			return;
		}
		busy = false;
		error = result.message;
	}

	onMount(run);
</script>

<svelte:head>
	<title>Signing you out</title>
</svelte:head>

<h1>Signing you out</h1>
{#if error}
	<p role="alert" data-testid="sign-out-error">{error}</p>
	<Button onclick={run} disabled={busy} data-testid="sign-out-retry">Try again</Button>
{:else}
	<p role="status">One moment.</p>
{/if}
