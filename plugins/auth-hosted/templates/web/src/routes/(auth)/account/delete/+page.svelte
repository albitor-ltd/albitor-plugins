<!-- Seeded by Albitor (auth-hosted), then yours: the confirmation before an account
     is deleted. On success the API has deleted the account and cleared the session, and the person
     goes through Cognito's sign-out page back to the app's home page. -->
<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { deleteAccount, isHostedUrl } from '../../../../lib/auth/client';
	import { getSession } from '../../../../lib/auth/session';
	import { Button } from '../../../../lib/kit';

	let email = $state('');
	let error = $state('');
	let busy = $state(false);
	let alertBox: HTMLElement | undefined = $state();

	onMount(async () => {
		email = (await getSession())?.email ?? '';
	});

	// The button is disabled while busy, which drops focus to <body>; a failure takes it back, on the alert.
	async function fail(message: string) {
		error = message;
		await tick();
		alertBox?.focus();
	}

	async function confirmDelete() {
		error = '';
		busy = true;
		const result = await deleteAccount();
		busy = false;
		if (!result.ok) {
			void fail(result.message);
			return;
		}
		if (isHostedUrl(result.logout_url)) {
			window.location.assign(result.logout_url);
			return;
		}
		await goto('/');
	}
</script>

<svelte:head>
	<title>Delete your account</title>
</svelte:head>

<h1>Are you sure you want to delete your account?</h1>
<p>
	You will be signed out and will not be able to sign in with {email || 'this email address'} again.
</p>
{#if error}
	<p role="alert" tabindex="-1" bind:this={alertBox} data-testid="delete-error">{error}</p>
{/if}
<p>
	<Button variant="destructive" onclick={confirmDelete} disabled={busy} data-testid="delete-confirm">
		Yes, delete my account
	</Button>
	<a href="/account" data-testid="delete-cancel">Cancel</a>
</p>
