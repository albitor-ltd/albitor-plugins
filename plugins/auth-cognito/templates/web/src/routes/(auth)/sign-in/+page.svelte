<!-- Seeded by Albitor (auth-cognito, kit-shadcn), then yours. Restyle and reword freely, but keep the
     data-testid values: the journey harness signs in through sign-in-form, sign-in-email,
     sign-in-password and sign-in-button. An invited person's first sign-in (the temporary password
     the pool emailed) swaps the form for new-password-form, new-password-input and
     new-password-button, where they choose their own. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { nextPath } from '../../../lib/auth/guard';
	import { newPassword, signIn } from '../../../lib/auth/client';
	import { Button, Field } from '../../../lib/kit';

	let email = $state(page.url.searchParams.get('email') ?? '');
	let password = $state('');
	let error = $state('');
	let busy = $state(false);
	// Set once the pool asks an invited person for their own password; held in memory only.
	let challengeSession = $state('');
	let chosenPassword = $state('');

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		busy = true;
		const result = await signIn(email, password);
		busy = false;
		if (result.ok) {
			await goto(nextPath(page.url.searchParams.get('next')));
			return;
		}
		password = '';
		if (result.code === 'new_password_required' && result.session) {
			challengeSession = result.session;
			return;
		}
		if (result.code === 'not_confirmed') {
			await goto(`/confirm?email=${encodeURIComponent(email)}`);
			return;
		}
		error = result.message;
	}

	async function submitNewPassword(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		busy = true;
		const result = await newPassword(email, chosenPassword, challengeSession);
		busy = false;
		if (result.ok) {
			await goto(nextPath(page.url.searchParams.get('next')));
			return;
		}
		chosenPassword = '';
		if (result.code === 'session_expired') challengeSession = '';
		error = result.message;
	}
</script>

<svelte:head>
	<title>Sign in</title>
</svelte:head>

<h1>{challengeSession ? 'Choose your password' : 'Sign in'}</h1>

{#if challengeSession}
	<form data-testid="new-password-form" onsubmit={submitNewPassword} novalidate>
		<p>Choose a password of your own to finish signing in.</p>
		<Field
			label="New password"
			type="password"
			autocomplete="new-password"
			required
			bind:value={chosenPassword}
			data-testid="new-password-input"
		/>
		{#if error}
			<p role="alert" data-testid="new-password-error">{error}</p>
		{/if}
		<Button type="submit" disabled={busy} data-testid="new-password-button">Set password and sign in</Button>
	</form>
{:else}
<form data-testid="sign-in-form" onsubmit={submit} novalidate>
	<Field
		label="Email"
		type="email"
		autocomplete="email"
		required
		bind:value={email}
		data-testid="sign-in-email"
	/>
	<Field
		label="Password"
		type="password"
		autocomplete="current-password"
		required
		bind:value={password}
		data-testid="sign-in-password"
	/>
	{#if error}
		<p role="alert" data-testid="sign-in-error">{error}</p>
	{/if}
	<Button type="submit" disabled={busy} data-testid="sign-in-button">Sign in</Button>
</form>
{/if}

<p>
	<a href="/sign-up" data-testid="sign-in-sign-up-link">Create an account</a>
	<span aria-hidden="true">·</span>
	<a href="/reset" data-testid="sign-in-reset-link">Forgot your password?</a>
</p>
