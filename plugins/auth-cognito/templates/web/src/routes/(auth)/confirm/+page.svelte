<!-- Seeded by Albitor (auth-cognito, component-kit@2), then yours. Keep the data-testid values. -->
<script lang="ts">
	import { tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { confirm, resend } from '../../../lib/auth/client';
	import { Button, Field } from '../../../lib/kit';

	let email = $state(page.url.searchParams.get('email') ?? '');
	let code = $state('');
	let error = $state('');
	let notice = $state('');
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
		error = notice = '';
		busy = true;
		const result = await confirm(email, code);
		busy = false;
		if (!result.ok) {
			void fail(result.message);
			return;
		}
		await goto(`/sign-in?email=${encodeURIComponent(email)}`);
	}

	async function sendAgain() {
		error = notice = '';
		const result = await resend(email);
		if (result.ok) notice = 'A new code is on its way.';
		else void fail(result.message);
	}
</script>

<svelte:head>
	<title>Confirm your email</title>
</svelte:head>

<h1>Confirm your email</h1>
<p>Enter the code we sent to your email address.</p>

<form data-testid="confirm-form" onsubmit={submit} novalidate>
	<Field
		label="Email"
		type="email"
		autocomplete="email"
		required
		bind:value={email}
		data-testid="confirm-email"
	/>
	<Field
		label="Code"
		inputmode="numeric"
		autocomplete="one-time-code"
		required
		bind:value={code}
		data-testid="confirm-code"
	/>
	{#if error}
		<p role="alert" tabindex="-1" bind:this={alertBox} data-testid="confirm-error">{error}</p>
	{/if}
	{#if notice}
		<p role="status" data-testid="confirm-notice">{notice}</p>
	{/if}
	<Button type="submit" disabled={busy} data-testid="confirm-button">Confirm</Button>
	<Button variant="secondary" onclick={sendAgain} data-testid="confirm-resend">Send a new code</Button>
</form>

<p><a href="/sign-in" data-testid="confirm-sign-in-link">Back to sign in</a></p>
