<!-- Seeded by Albitor (auth-cognito, kit-shadcn), then yours. Restyle and reword freely, but keep the
     data-testid values: the journey harness signs in through sign-in-form, sign-in-email,
     sign-in-password and sign-in-button. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { nextPath } from '../../../lib/auth/guard';
	import { signIn } from '../../../lib/auth/client';
	import { Button, Field } from '../../../lib/kit';

	let email = $state(page.url.searchParams.get('email') ?? '');
	let password = $state('');
	let error = $state('');
	let busy = $state(false);

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
		if (result.code === 'not_confirmed') {
			await goto(`/confirm?email=${encodeURIComponent(email)}`);
			return;
		}
		error = result.message;
	}
</script>

<svelte:head>
	<title>Sign in</title>
</svelte:head>

<h1>Sign in</h1>

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

<p>
	<a href="/sign-up" data-testid="sign-in-sign-up-link">Create an account</a>
	<span aria-hidden="true">·</span>
	<a href="/reset" data-testid="sign-in-reset-link">Forgot your password?</a>
</p>
