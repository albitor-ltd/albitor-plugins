<!-- Seeded by Albitor (kit-carbon), then yours: a modal dialog on Carbon's Modal, which owns the
     focus trap, Escape, the overlay and returning focus to the trigger. Always give it a title and a
     description: the title is the dialog's accessible name and the description is read with it.
     Carbon keeps a closed Modal in the page (inert), so the dialog role is always present; the
     open state is the `is-visible` class on its wrapper. -->
<script lang="ts">
	import { Modal } from 'carbon-components-svelte';
	import type { Snippet } from 'svelte';

	let {
		open = $bindable(false),
		title,
		description,
		children,
		actions
	}: {
		open?: boolean;
		title: string;
		description: string;
		children?: Snippet;
		actions?: Snippet;
	} = $props();

	const uid = $props.id();
	const descriptionId = `${uid}-description`;
	let wrapper: HTMLDivElement | null = $state(null);

	// Carbon names the dialog from its heading but only describes an alert dialog. Point the dialog
	// at the description so it is announced with the title.
	$effect(() => {
		wrapper?.querySelector('[role="dialog"]')?.setAttribute('aria-describedby', descriptionId);
	});
</script>

<Modal
	bind:open
	bind:ref={wrapper}
	passiveModal
	modalHeading={title}
	iconDescription="Close"
>
	<p id={descriptionId} class="description">{description}</p>
	{#if children}{@render children()}{/if}
	<div class="actions">
		{#if actions}{@render actions()}{/if}
	</div>
</Modal>

<style>
	.description {
		margin: 0 0 1rem;
		color: var(--cds-text-secondary);
	}
	.actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.5rem;
		margin-top: 1.5rem;
	}
</style>
