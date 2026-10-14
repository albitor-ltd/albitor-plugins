<!-- Seeded by Albitor (kit-shadcn), then yours: a modal dialog on Bits UI, which owns the focus trap,
     Escape, scroll lock and returning focus to the trigger. Always give it a title: the title is
     the dialog's accessible name. -->
<script lang="ts">
	import { Dialog } from 'bits-ui';
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
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="kit-overlay" />
		<Dialog.Content class="kit-dialog">
			<Dialog.Title class="kit-dialog-title">{title}</Dialog.Title>
			<Dialog.Description class="kit-dialog-description">{description}</Dialog.Description>
			{#if children}{@render children()}{/if}
			<div class="kit-dialog-actions">
				{#if actions}{@render actions()}{/if}
				<Dialog.Close class="kit-dialog-close">Close</Dialog.Close>
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<style>
	/* Bits UI renders into a portal outside this component, so these are global by necessity. */
	:global(.kit-overlay) {
		position: fixed;
		inset: 0;
		background: rgb(0 0 0 / 0.6);
		z-index: 30;
	}
	:global(.kit-dialog) {
		position: fixed;
		inset-block-start: 50%;
		inset-inline-start: 50%;
		transform: translate(-50%, -50%);
		width: min(32rem, calc(100vw - 2rem));
		max-height: calc(100vh - 2rem);
		overflow: auto;
		box-sizing: border-box;
		padding: 1.5rem;
		background: var(--background);
		color: var(--foreground);
		border: 1px solid var(--input);
		border-radius: var(--radius);
		z-index: 31;
	}
	:global(.kit-dialog-title) {
		margin: 0 0 0.5rem;
		font-size: var(--text-heading);
		font-weight: 600;
	}
	:global(.kit-dialog-description) {
		margin: 0 0 1rem;
		color: var(--muted-foreground);
	}
	:global(.kit-dialog-actions) {
		display: flex;
		justify-content: flex-end;
		gap: 0.5rem;
		margin-top: 1.5rem;
	}
	:global(.kit-dialog-close) {
		min-height: 2.5rem;
		padding: 0 1rem;
		background: var(--background);
		color: var(--foreground);
		border: 1px solid var(--input);
		border-radius: var(--radius);
		font: inherit;
		cursor: pointer;
	}
	:global(.kit-dialog-close:focus-visible) {
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}
</style>
