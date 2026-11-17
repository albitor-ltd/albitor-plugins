<!-- Keep the data-testid values. -->
<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { confirm, resend } from '../../../lib/auth/client';
	import { Button, Field } from '../../../lib/kit';

	let email = $state(page.url.searchParams.get('email') ?? '');
	let code = $state('');
	let error = $state('');
	let notice = $state('');
	let busy = $state(false);
	let alreadyConfirmed = $state(false);
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
		alreadyConfirmed = false;
		const result = await resend(email);
		if (result.ok) {
			notice = `If an account with ${email} is waiting for confirmation, we've sent it a new code.`;
		} else if (result.code === 'already_confirmed') {
			alreadyConfirmed = true;
		} else {
			void fail(result.message);
		}
	}

	// Arriving with resend=1 (from a sign-up or sign-in that met an unconfirmed account) sends a new
	// code once. onMount runs once per visit, and the parameter is dropped so a reload does not
	// send another.
	onMount(() => {
		if (page.url.searchParams.get('resend') !== '1' || !email) return;
		const url = new URL(window.location.href);
		url.searchParams.delete('resend');
		history.replaceState(history.state, '', url);
		void sendAgain();
	});
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
	{#if alreadyConfirmed}
		<p role="status" data-testid="confirm-already">
			This email is already confirmed. <a href="/sign-in?email={encodeURIComponent(email)}">Sign in</a>
		</p>
	{/if}
	{#if notice}
		<p role="status" data-testid="confirm-notice">{notice}</p>
	{/if}
	<Button type="submit" disabled={busy} data-testid="confirm-button">Confirm</Button>
	<Button variant="secondary" onclick={sendAgain} data-testid="confirm-resend">Send a new code</Button>
</form>

<p>
	Already confirmed?
	<a href="/sign-in?email={encodeURIComponent(email)}" data-testid="confirm-sign-in-link">Sign in</a>
</p>
