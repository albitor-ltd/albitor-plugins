<!-- Seeded by Albitor (auth-cognito, component-kit@2), then yours. Keep the data-testid values. Add the
     sign-up fields the brief asks for here and in the API call (client.ts). -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { signUp } from '../../../lib/auth/client';
	import { Button, Field } from '../../../lib/kit';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let busy = $state(false);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		busy = true;
		const result = await signUp(email, password);
		busy = false;
		if (!result.ok) {
			error = result.message;
			return;
		}
		await goto(
			result.confirmation_required
				? `/confirm?email=${encodeURIComponent(email)}`
				: `/sign-in?email=${encodeURIComponent(email)}`
		);
	}
</script>

<svelte:head>
	<title>Create an account</title>
</svelte:head>

<h1>Create an account</h1>

<form data-testid="sign-up-form" onsubmit={submit} novalidate>
	<Field
		label="Email"
		type="email"
		autocomplete="email"
		required
		bind:value={email}
		data-testid="sign-up-email"
	/>
	<Field
		label="Password"
		type="password"
		autocomplete="new-password"
		hint="At least 12 characters, with a lower-case letter, an upper-case letter and a number."
		required
		bind:value={password}
		data-testid="sign-up-password"
	/>
	{#if error}
		<p role="alert" data-testid="sign-up-error">{error}</p>
	{/if}
	<Button type="submit" disabled={busy} data-testid="sign-up-button">Create account</Button>
</form>

<p>
	Already have an account?
	<a href="/sign-in" data-testid="sign-up-sign-in-link">Sign in</a>
</p>
