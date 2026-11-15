<!-- Notifications: the optional and consent emails this app sends, each with the reason it is
     sent and a switch. Optional email starts on, consent email starts off, and a save sends only
     what the person changed. It sits under Settings and is linked from there (settings/+page.svelte)
     and from the nav (settings/nav.ts); keep the route at /settings/notifications, which the
     check of the finished app looks for. Keep the data-testid values. -->
<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { Button, toast } from '../../../../lib/kit';
	import {
		changedChoices,
		loadNotifications,
		saveNotifications,
		type Notification
	} from '../../../../lib/notifications/client';

	let loaded = $state<Notification[]>([]);
	let now = $state<Record<string, boolean>>({});
	let loading = $state(true);
	let busy = $state(false);
	let error = $state('');
	let alertBox: HTMLElement | undefined = $state();

	const optional = $derived(loaded.filter((n) => !n.consent));
	const consent = $derived(loaded.filter((n) => n.consent));
	const changed = $derived(changedChoices(loaded, now));
	const dirty = $derived(Object.keys(changed).length > 0);

	async function fail(message: string) {
		error = message;
		await tick();
		alertBox?.focus();
	}

	onMount(async () => {
		const result = await loadNotifications();
		loading = false;
		if (!result.ok) return void fail(result.message);
		loaded = result.notifications;
		now = Object.fromEntries(loaded.map((n) => [n.template_id, n.subscribed]));
	});

	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (busy || !dirty) return;
		busy = true;
		error = '';
		const result = await saveNotifications(changed);
		busy = false;
		if (!result.ok) return void fail(result.message);
		loaded = loaded.map((n) => ({ ...n, subscribed: now[n.template_id] ?? n.subscribed }));
		toast.push({ message: 'Email settings saved', tone: 'success' });
	}
</script>

<svelte:head>
	<title>Notifications</title>
</svelte:head>

<h1>Notifications</h1>
<p>Choose which emails this app sends you. Emails that are needed to use the app, such as a
	confirmation you asked for, are not listed here.</p>

{#if error}
	<div role="alert" tabindex="-1" bind:this={alertBox} data-testid="notifications-error">{error}</div>
{/if}

{#if loading}
	<p>Loading…</p>
{:else if loaded.length === 0 && !error}
	<p data-testid="notifications-none">This app has no optional emails to choose about.</p>
{:else if loaded.length > 0}
	<form onsubmit={save} novalidate>
		{#if optional.length > 0}
			<fieldset data-testid="notifications-optional">
				<legend>Emails you can turn off</legend>
				{#each optional as n (n.template_id)}
					<div>
						<label>
							<input type="checkbox" bind:checked={now[n.template_id]} data-testid={`notification-${n.template_id}`} />
							{n.name}
						</label>
						<p class="reason" data-testid={`reason-${n.template_id}`}>{n.reason}</p>
					</div>
				{/each}
			</fieldset>
		{/if}
		{#if consent.length > 0}
			<fieldset data-testid="notifications-consent">
				<legend>Emails we only send if you agree</legend>
				{#each consent as n (n.template_id)}
					<div>
						<label>
							<input type="checkbox" bind:checked={now[n.template_id]} data-testid={`notification-${n.template_id}`} />
							{n.name}
						</label>
						<p class="reason" data-testid={`reason-${n.template_id}`}>{n.reason}</p>
					</div>
				{/each}
			</fieldset>
		{/if}
		<Button type="submit" disabled={busy || !dirty} data-testid="notifications-save">
			{busy ? 'Saving…' : 'Save changes'}
		</Button>
	</form>
{/if}

<style>
	fieldset {
		margin: 0 0 1.5rem;
		max-width: 40rem;
	}
	.reason {
		margin: 0.25rem 0 1rem 1.75rem;
		color: var(--muted-foreground);
	}
	[role='alert'] {
		max-width: 40rem;
		margin-bottom: 1rem;
		padding: 0.75rem 1rem;
		border: 2px solid var(--destructive);
		border-radius: var(--radius);
	}
</style>
