<!-- Seeded by Albitor (kit-shadcn), then yours: an overview. A header, a row of headline figures,
     then sections of detail as children (a Table, a short list, a chart with a text or table
     equivalent). Every figure is text with a label, so a screen reader reads "Open items 12", and
     nothing is conveyed by colour alone.

       <DashboardPage title="Overview" stats={[{ label: 'Open items', value: '12' }]} {status}>
         <h2>Recent activity</h2> <Table ... />
       </DashboardPage> -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import PageHeader from './PageHeader.svelte';
	import StateNotice from './StateNotice.svelte';

	let {
		title,
		description,
		stats,
		status = 'ready',
		errorMessage,
		onretry,
		actions,
		children
	}: {
		title: string;
		description?: string;
		stats: Array<{ label: string; value: string | number; hint?: string }>;
		status?: 'loading' | 'error' | 'ready';
		errorMessage?: string;
		onretry?: () => void;
		actions?: Snippet;
		/** Sections below the figures: give each its own `<h2>`. */
		children?: Snippet;
	} = $props();
</script>

<PageHeader {title} {description} {actions} />

{#if status !== 'ready'}
	<StateNotice {status} {errorMessage} {onretry} loadingLabel={`Loading ${title.toLowerCase()}`} />
{:else}
	<dl class="stats">
		{#each stats as stat (stat.label)}
			<div class="stat">
				<dt>{stat.label}</dt>
				<dd>{stat.value}</dd>
				{#if stat.hint}<dd class="hint">{stat.hint}</dd>{/if}
			</div>
		{/each}
	</dl>
	{#if children}<div class="sections">{@render children()}</div>{/if}
{/if}

<style>
	.stats {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
		gap: 1rem;
		margin: 0 0 2rem;
	}
	.stat {
		padding: 1rem;
		border: 1px solid var(--border);
		border-radius: var(--radius);
	}
	dt {
		color: var(--muted-foreground);
		font-size: var(--text-small);
	}
	dd {
		margin: 0.25rem 0 0;
		font-size: var(--text-title);
		font-weight: 600;
	}
	dd.hint {
		color: var(--muted-foreground);
		font-size: var(--text-small);
		font-weight: 400;
	}
	.sections :global(h2) {
		font-size: var(--text-heading);
	}
</style>
