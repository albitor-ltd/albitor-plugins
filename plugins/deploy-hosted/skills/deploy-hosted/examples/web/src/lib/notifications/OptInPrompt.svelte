<!-- The first-sign-in opt-in. Put it on the home page, once, for an app that has consent email: it
     shows only until the person answers or closes it, once per person on this browser, and the
     box starts unticked. Ticking it and pressing "Save my choice" is the person's own act, which
     is what the platform records as their consent; closing it records nothing. The Notifications
     page stays the place to change the choice later.

       <OptInPrompt who={session.sub} /> -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { Button } from '../kit';
	import { loadNotifications, saveNotifications, type Notification } from './client';
	import { consentToAsk, markAsked, shouldAsk } from './optin';

	let { who }: { who: string } = $props();

	let ask = $state<Notification[]>([]);
	let ticked = $state<Record<string, boolean>>({});
	let busy = $state(false);
	let error = $state('');

	onMount(async () => {
		const result = await loadNotifications();
		if (!result.ok || !shouldAsk(result.notifications, who)) return;
		ask = consentToAsk(result.notifications);
	});

	function close() {
		markAsked(who);
		ask = [];
	}

	async function save() {
		busy = true;
		error = '';
		const chosen = Object.fromEntries(Object.entries(ticked).filter(([, on]) => on));
		const result = await saveNotifications(chosen);
		busy = false;
		if (!result.ok) {
			error = result.message;
			return;
		}
		close();
	}
</script>

{#if ask.length > 0}
	<section aria-labelledby="optin-title" data-testid="optin-prompt">
		<h2 id="optin-title">Would you like these emails?</h2>
		<p>We only send them if you agree. You can change this any time under Settings, then Notifications.</p>
		{#each ask as n (n.template_id)}
			<label>
				<input type="checkbox" bind:checked={ticked[n.template_id]} data-testid={`optin-${n.template_id}`} />
				{n.name}
			</label>
			<p>{n.reason}</p>
		{/each}
		{#if error}<p role="alert">{error}</p>{/if}
		<Button onclick={save} disabled={busy} data-testid="optin-save">Save my choice</Button>
		<Button variant="secondary" onclick={close} data-testid="optin-close">Not now</Button>
	</section>
{/if}
