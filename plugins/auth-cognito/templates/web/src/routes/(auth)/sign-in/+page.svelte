<!-- Seeded by Albitor (auth-cognito, component-kit@2), then yours. The sign-in form is the kit's
     `SignInPage` (the same props in every kit), so it takes the kit's look and its error handling.
     Restyle and reword freely. `?email=` prefills the address. A person's first sign-in with the temporary password the pool
     emailed swaps it for a short form of kit `Field`s and `Button`, where they choose their own
     password (data-testid new-password-form, new-password-input, new-password-button). -->
<script lang="ts">
	import { tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { nextPath } from '../../../lib/auth/guard';
	import { newPassword, signIn } from '../../../lib/auth/client';
	import { Button, Field, SignInPage } from '../../../lib/kit';

	// The typed address is kept here (SignInPage starts from it), so it survives the form being
	// remounted after an expired new-password step. `?email=` seeds it, as sign-up and confirm send it.
	let email = $state(page.url.searchParams.get('email') ?? '');
	let error = $state('');
	let busy = $state(false);
	// Set once the pool asks an invited person for their own password; held in memory only.
	let challengeSession = $state('');
	let chosenPassword = $state('');
	let heading: HTMLHeadingElement | undefined = $state();

	// The form is swapped for another in place, so focus would be left on a removed button and a
	// screen-reader or keyboard user would not know the page changed (WCAG 2.4.3, 3.2.2). Moving it
	// to the new heading announces the step without jumping to a field they have not read about.
	async function announceStep() {
		await tick();
		heading?.focus();
	}

	async function submit(enteredEmail: string, password: string) {
		email = enteredEmail;
		error = '';
		busy = true;
		const result = await signIn(email, password);
		busy = false;
		if (result.ok) {
			await goto(nextPath(page.url.searchParams.get('next')));
			return;
		}
		if (result.code === 'new_password_required' && result.session) {
			challengeSession = result.session;
			await announceStep();
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
		if (result.code === 'session_expired') {
			challengeSession = '';
			error = result.message;
			await announceStep();
			return;
		}
		error = result.message;
	}
</script>

<svelte:head>
	<title>Sign in</title>
</svelte:head>

{#if challengeSession}
	<h1 bind:this={heading} tabindex="-1">Choose your password</h1>
	<form data-testid="new-password-form" onsubmit={submitNewPassword} novalidate>
		<p>Choose a password of your own to finish signing in.</p>
		<Field
			label="New password"
			type="password"
			autocomplete="new-password"
			hint="At least 12 characters, with a lower-case letter, an upper-case letter and a number."
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
	<SignInPage {email} {error} {busy} onsubmit={submit} resetHref="/reset" signUpHref="/sign-up" />
{/if}
