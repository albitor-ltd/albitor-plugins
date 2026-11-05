<!-- A signed-in person deletes their own account (both sign-up modes). Every app needs this: a
     single-role app has no administrator to do it for them. The account is removed from the pool,
     so the same email and password no longer sign in; the data the app holds for them is removed by
     the app's OnAccountDeleted hook (see identity_people.go). Reached from the change-password
     screen; keep the data-testid values. -->
<script lang="ts">
	import { tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { deleteAccount } from '../../../lib/auth/client';
	import { Button, Field } from '../../../lib/kit';

	let password = $state('');
	let error = $state('');
	let busy = $state(false);
	let alertBox: HTMLElement | undefined = $state();

	// The button is disabled while busy, which drops focus to <body>; a failure takes it back, on the alert.
	async function fail(message: string) {
		error = message;
		await tick();
		alertBox?.focus();
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		if (password === '') {
			void fail('Enter your password to confirm.');
			return;
		}
		busy = true;
		const result = await deleteAccount(password);
		busy = false;
		if (!result.ok) {
			void fail(result.message);
			return;
		}
		password = '';
		await goto('/sign-in');
	}
</script>

<svelte:head>
	<title>Delete your account</title>
</svelte:head>

<h1>Are you sure you want to delete your account?</h1>
<p>You will be signed out and will not be able to sign in with this email address again.</p>

<form data-testid="delete-account-form" onsubmit={submit} novalidate>
	<Field
		label="Your password"
		type="password"
		autocomplete="current-password"
		required
		bind:value={password}
		data-testid="delete-account-password"
	/>
	{#if error}
		<p role="alert" tabindex="-1" bind:this={alertBox} data-testid="delete-account-error">{error}</p>
	{/if}
	<Button type="submit" variant="destructive" disabled={busy} data-testid="delete-account-button">
		Yes, delete my account
	</Button>
	<a href="/change-password" data-testid="delete-account-cancel">Cancel</a>
</form>
