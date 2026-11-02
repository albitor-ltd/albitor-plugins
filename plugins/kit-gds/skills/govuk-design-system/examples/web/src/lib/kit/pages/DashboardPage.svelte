<!-- An overview. A header, a row of headline figures, then sections of detail as children (a
     Table, a short list, a chart with a text or table equivalent). Every figure is text with a
     label, so a screen reader reads "Open items 12", and nothing is conveyed by colour alone.

       <DashboardPage title="Overview" stats={[{ label: 'Open items', value: '12' }]} {status}>
         <h2 class="govuk-heading-m">Recent activity</h2> <Table ... />
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
		/** Sections below the figures: give each its own `<h2 class="govuk-heading-m">`. */
		children?: Snippet;
	} = $props();
</script>

<PageHeader {title} {description} {actions} />

{#if status !== 'ready'}
	<StateNotice {status} {errorMessage} {onretry} loadingLabel={`Loading ${title.toLowerCase()}`} />
{:else}
	<dl class="govuk-grid-row stats">
		{#each stats as stat (stat.label)}
			<div class="govuk-grid-column-one-third stat">
				<dt class="govuk-body-s">{stat.label}</dt>
				<dd class="govuk-heading-l">{stat.value}</dd>
				{#if stat.hint}<dd class="govuk-body-s">{stat.hint}</dd>{/if}
			</div>
		{/each}
	</dl>
	{#if children}<div>{@render children()}</div>{/if}
{/if}

<style>
	.stats {
		margin-bottom: 30px;
	}
	.stat::before {
		display: block;
		margin-bottom: 10px;
		border-top: 5px solid currentColor;
		content: '';
	}
	.stat dt {
		margin: 0;
	}
	.stat dd {
		margin: 0 0 5px;
	}
</style>
