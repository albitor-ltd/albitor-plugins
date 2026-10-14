<!-- Seeded by Albitor (kit-shadcn), then yours: render once, near the root. Messages come from
     toast.push(). Nothing disappears on a timer, so nobody is cut off mid-read (WCAG 2.2.1); each
     toast has a dismiss button. Both live regions are always in the page, because a region
     inserted together with its message is often not announced. -->
<script lang="ts">
	import { toast } from './toast.svelte';

	const polite = $derived(toast.items.filter((t) => t.tone !== 'error'));
	const urgent = $derived(toast.items.filter((t) => t.tone === 'error'));
</script>

<div class="toaster">
	<div role="status">
		{#each polite as t (t.id)}
			<p class={t.tone}>
				<span>{t.message}</span>
				<button type="button" onclick={() => toast.dismiss(t.id)}>
					Dismiss<span class="sr">: {t.message}</span>
				</button>
			</p>
		{/each}
	</div>
	<div role="alert">
		{#each urgent as t (t.id)}
			<p class={t.tone}>
				<span>{t.message}</span>
				<button type="button" onclick={() => toast.dismiss(t.id)}>
					Dismiss<span class="sr">: {t.message}</span>
				</button>
			</p>
		{/each}
	</div>
</div>

<style>
	.toaster {
		position: fixed;
		inset-block-end: 1rem;
		inset-inline-end: 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		max-width: min(24rem, calc(100vw - 2rem));
		z-index: 20;
	}
	p {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		margin: 0 0 0.5rem;
		padding: 0.75rem 1rem;
		background: var(--background);
		color: var(--foreground);
		border: 1px solid var(--input);
		border-inline-start-width: 6px;
		border-radius: var(--radius);
	}
	p.success {
		border-inline-start-color: var(--link);
	}
	p.error {
		border-color: var(--destructive);
	}
	button {
		min-height: 2rem;
		padding: 0 0.75rem;
		background: transparent;
		color: inherit;
		border: 1px solid var(--input);
		border-radius: var(--radius);
		font: inherit;
		cursor: pointer;
	}
	button:focus-visible {
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}
	.sr {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}
</style>
