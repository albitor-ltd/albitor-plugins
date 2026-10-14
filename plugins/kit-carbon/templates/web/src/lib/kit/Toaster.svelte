<!-- Seeded by Albitor (kit-carbon), then yours: render once, near the root. Messages come from
     toast.push(). Nothing disappears on a timer, so nobody is cut off mid-read (WCAG 2.2.1); each
     toast has a dismiss button. Both live regions are always in the page, because a region
     inserted together with its message is often not announced. The toasts use Carbon's toast
     classes and icon and close button, not its ToastNotification: that one sets its own alert role
     and puts the message in a heading, which would announce twice and break the heading order. -->
<script lang="ts">
	import { NotificationButton, NotificationIcon } from 'carbon-components-svelte';
	import { toast, type Toast } from './toast.svelte';

	const polite = $derived(toast.items.filter((t) => t.tone !== 'error'));
	const urgent = $derived(toast.items.filter((t) => t.tone === 'error'));
</script>

{#snippet item(t: Toast)}
	<div class="bx--toast-notification bx--toast-notification--low-contrast bx--toast-notification--{t.tone}">
		<NotificationIcon notificationType="toast" kind={t.tone} />
		<div class="bx--toast-notification__details">
			<p class="bx--toast-notification__title">{t.message}</p>
		</div>
		<NotificationButton
			notificationType="toast"
			iconDescription="Dismiss: {t.message}"
			on:click={() => toast.dismiss(t.id)}
		/>
	</div>
{/snippet}

<div class="toaster">
	<div role="status">
		{#each polite as t (t.id)}{@render item(t)}{/each}
	</div>
	<div role="alert">
		{#each urgent as t (t.id)}{@render item(t)}{/each}
	</div>
</div>

<style>
	.toaster {
		position: fixed;
		inset-block-end: 1rem;
		inset-inline-end: 1rem;
		max-width: min(24rem, calc(100vw - 2rem));
		z-index: 9000;
	}
	.toaster :global(.bx--toast-notification) {
		width: 100%;
		margin: 0 0 0.5rem;
	}
</style>
