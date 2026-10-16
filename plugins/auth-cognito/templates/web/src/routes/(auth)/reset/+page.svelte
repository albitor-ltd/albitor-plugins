<!-- Seeded by Albitor (auth-cognito, kit-shadcn), then yours. Two steps on one page: ask for a code,
     then set a new password with it. Keep the data-testid values. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { forgot, reset } from '../../../lib/auth/client';
	import { Button, Field } from '../../../lib/kit';

	let email = $state('');
	let code = $state('');
	let password = $state('');
	let sent = $state(false);
	let error = $state('');
	let busy = $state(false);

	async function request(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		busy = true;
		const result = await forgot(email);
		busy = false;
		if (result.ok) sent = true;
		else error = result.message;
	}

	async function finish(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		busy = true;
		const result = await reset(email, code, password);
		busy = false;
		if (!result.ok) {
			error = result.message;
			return;
		}
		await goto(`/sign-in?email=${encodeURIComponent(email)}`);
	}
</script>

<svelte:head>
	<title>Reset your password</title>
</svelte:head>

<h1>Reset your password</h1>

{#if !sent}
	<form data-testid="reset-request-form" onsubmit={request} novalidate>
		<Field
			label="Email"
			type="email"
			autocomplete="email"
			required
			bind:value={email}
			data-testid="reset-email"
		/>
		{#if error}
			<p role="alert" data-testid="reset-error">{error}</p>
		{/if}
		<Button type="submit" disabled={busy} data-testid="reset-request-button">Send a code</Button>
	</form>
{:else}
	<p role="status">If that address has an account, a code is on its way.</p>
	<form data-testid="reset-form" onsubmit={finish} novalidate>
		<Field
			label="Code"
			inputmode="numeric"
			autocomplete="one-time-code"
			required
			bind:value={code}
			data-testid="reset-code"
		/>
		<Field
			label="New password"
			type="password"
			autocomplete="new-password"
			hint="At least 12 characters, with a lower-case letter, an upper-case letter and a number."
			required
			bind:value={password}
			data-testid="reset-password"
		/>
		{#if error}
			<p role="alert" data-testid="reset-error">{error}</p>
		{/if}
		<Button type="submit" disabled={busy} data-testid="reset-button">Set new password</Button>
	</form>
{/if}

<p><a href="/sign-in" data-testid="reset-sign-in-link">Back to sign in</a></p>
