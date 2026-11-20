<!-- The page after a flow: it worked, here is the reference, here is what happens next. The headline
     states the result ("Application sent"), a reference the person can quote sits in a Card, and
     the next steps are a short list. It is a page, not a toast: a person who looks away still finds
     it, and it has a link onward.

       <ConfirmationPage title="Application sent" reference="PRM-48213"
                         next={['We email you within 3 working days.']} action={{ href: '/', label: 'Back to your account' }} /> -->
<script lang="ts">
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import Card from '../Card.svelte';
	import Icon from '../Icon.svelte';
	import ActionLink from './ActionLink.svelte';

	let {
		title,
		reference,
		referenceLabel = 'Your reference',
		next = [],
		action
	}: {
		title: string;
		reference?: string;
		referenceLabel?: string;
		/** What happens next, one sentence each. */
		next?: string[];
		/** The way onward. */
		action: { href: string; label: string };
	} = $props();
</script>

<div class="done">
	<h1><Icon icon={CircleCheck} size={28} /> {title}</h1>
	{#if reference}
		<Card tone="second">
			<p class="ref"><span>{referenceLabel}</span> <strong>{reference}</strong></p>
		</Card>
	{/if}
	{#if next.length > 0}
		<h2>What happens next</h2>
		<ul>
			{#each next as line (line)}<li>{line}</li>{/each}
		</ul>
	{/if}
	<ActionLink href={action.href}>{action.label}</ActionLink>
</div>

<style>
	.done {
		max-width: 40rem;
		display: grid;
		gap: var(--space-4);
		justify-items: start;
	}
	h1 {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		margin: 0;
		font-size: var(--text-title);
	}
	h2 {
		margin: 0;
		font-size: var(--text-heading);
	}
	.done :global([data-kit='card']) {
		justify-self: stretch;
	}
	.ref {
		display: grid;
		gap: var(--space-1);
		margin: 0;
	}
	ul {
		margin: 0;
		padding-inline-start: 1.25rem;
	}
</style>
