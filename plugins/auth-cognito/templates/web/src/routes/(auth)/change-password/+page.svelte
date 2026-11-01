<!-- Seeded by Albitor (auth-cognito), then yours. A signed-in person changes their own password
     (both sign-up modes). Keep the data-testid values. Its nav.ts puts "Change password" in the
     nav; delete nav.ts to take it out. -->
<script lang="ts">
	import { tick } from 'svelte';
	import { onMount } from 'svelte';
	import { changePassword, getUserAdminGroup } from '../../../lib/auth/client';
	import { getSession } from '../../../lib/auth/session';
	import { Button, Field } from '../../../lib/kit';

	let current = $state('');
	let next = $state('');
	let confirmNext = $state('');
	let error = $state('');
	let done = $state(false);
	let busy = $state(false);
	let alertBox: HTMLElement | undefined = $state();
	// People is offered only to the members of the user-admin group (the nav is static, so the link
	// lives here, on the account screen).
	let canManagePeople = $state(false);
	onMount(async () => {
		const group = await getUserAdminGroup();
		canManagePeople = group !== '' && ((await getSession())?.groups ?? []).includes(group);
	});

	// The button is disabled while busy, which drops focus to <body>; a failure takes it back, on the alert.
	async function fail(message: string) {
		error = message;
		await tick();
		alertBox?.focus();
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		done = false;
		if (next !== confirmNext) {
			void fail('The new passwords do not match.');
			return;
		}
		busy = true;
		const result = await changePassword(current, next);
		busy = false;
		if (!result.ok) {
			void fail(result.message);
			return;
		}
		current = next = confirmNext = '';
		done = true;
	}
</script>

<svelte:head>
	<title>Change password</title>
</svelte:head>

<h1>Change password</h1>

{#if canManagePeople}
	<p><a href="/people" data-testid="people-link">Manage people</a></p>
{/if}

<form data-testid="change-password-form" onsubmit={submit} novalidate>
	<Field
		label="Current password"
		type="password"
		autocomplete="current-password"
		required
		bind:value={current}
		data-testid="change-password-current"
	/>
	<Field
		label="New password"
		type="password"
		autocomplete="new-password"
		hint="At least 12 characters, with a lower-case letter, an upper-case letter and a number."
		required
		bind:value={next}
		data-testid="change-password-new"
	/>
	<Field
		label="Confirm new password"
		type="password"
		autocomplete="new-password"
		required
		bind:value={confirmNext}
		data-testid="change-password-confirm"
	/>
	{#if error}
		<p role="alert" tabindex="-1" bind:this={alertBox} data-testid="change-password-error">{error}</p>
	{/if}
	{#if done}
		<p role="status" data-testid="change-password-done">Your password has been changed.</p>
	{/if}
	<Button type="submit" disabled={busy} data-testid="change-password-button">Change password</Button>
</form>
