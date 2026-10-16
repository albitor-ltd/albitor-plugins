<!-- Seeded by Albitor (kit-shadcn), then yours: the loading and failed states of a page, shown where
     the content will appear. Loading is a polite status; a failure is an alert with a retry,
     because a toast is gone once dismissed. -->
<script lang="ts">
	import Button from '../Button.svelte';

	let {
		status,
		loadingLabel = 'Loading',
		errorMessage = 'Something went wrong loading this page.',
		onretry
	}: {
		status: 'loading' | 'error';
		loadingLabel?: string;
		/** A sentence a person can act on. Never a raw server body. */
		errorMessage?: string;
		onretry?: () => void;
	} = $props();
</script>

{#if status === 'loading'}
	<p role="status" class="notice loading">{loadingLabel}…</p>
{:else}
	<div role="alert" class="notice">
		<p>{errorMessage}</p>
		{#if onretry}<Button variant="secondary" onclick={onretry}>Try again</Button>{/if}
	</div>
{/if}

<style>
	.notice {
		padding: 1rem;
		border: 1px solid var(--border);
		border-radius: var(--radius);
	}
	.notice p {
		margin: 0 0 0.75rem;
	}
	.loading {
		margin: 0;
	}
</style>
