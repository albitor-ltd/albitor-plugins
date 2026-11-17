<!-- The loading and failed states of a page, shown where
     the content will appear. Loading is a polite status with skeletons in the shape of the
     content; a failure is an alert with a retry, because a toast is gone once dismissed. -->
<script lang="ts">
	import Button from '../Button.svelte';
	import Skeleton from '../Skeleton.svelte';

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
	<div class="notice" aria-busy="true">
		<p role="status" class="loading">{loadingLabel}…</p>
		<div class="skeletons">
			<Skeleton height="1.25rem" width="40%" />
			<Skeleton height="1rem" />
			<Skeleton height="1rem" width="85%" />
		</div>
	</div>
{:else}
	<div role="alert" class="notice">
		<p>{errorMessage}</p>
		{#if onretry}<Button variant="secondary" onclick={onretry}>Try again</Button>{/if}
	</div>
{/if}

<style>
	.notice {
		padding: var(--space-4);
		background: var(--surface);
		border: var(--border-width) solid var(--border);
		border-radius: var(--radius);
		box-shadow: var(--shadow);
	}
	.notice p {
		margin: 0 0 var(--space-3);
	}
	.skeletons {
		display: grid;
		gap: var(--space-2);
	}
</style>
