<!-- An overview. A header, a row of headline figures (StatTile: an optional icon, a change in
     words, a meter or a sparkline), then sections of detail as children (a Card with a Table, a
     short list, a Chart). Every figure is text with a label, so a screen reader reads "Open items
     12", and nothing is conveyed by colour alone. Before the app has any data, pass `firstRun`: the
     page shows what to do first instead of a row of zeros.

       <DashboardPage title="Overview" stats={[{ label: 'Open items', value: '12' }]} {status}>
         <Card title="Recent activity"><Table ... /></Card>
       </DashboardPage> -->
<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import EmptyState from '../EmptyState.svelte';
	import StatTile from '../StatTile.svelte';
	import PageHeader from './PageHeader.svelte';
	import StateNotice from './StateNotice.svelte';

	let {
		title,
		description,
		stats,
		firstRun = false,
		firstRunTitle = 'Nothing to show yet',
		firstRunDescription,
		firstRunIcon,
		firstRunAction,
		status = 'ready',
		errorMessage,
		onretry,
		actions,
		children
	}: {
		title: string;
		description?: string;
		stats: Array<{
			label: string;
			value: string | number;
			icon?: Component<{ size?: number | string; class?: string }>;
			delta?: { direction: 'up' | 'down' | 'flat'; text: string; good?: boolean };
			meter?: { value: number; label: string };
			spark?: { data: Array<{ label: string; value: number }>; label: string };
		}>;
		/** No data yet: show what to do first in place of the figures and sections. */
		firstRun?: boolean;
		firstRunTitle?: string;
		/** Say why it is empty and what to do next. */
		firstRunDescription?: string;
		firstRunIcon?: Component<{ size?: number | string; class?: string }>;
		/** The button that starts the first item. */
		firstRunAction?: Snippet;
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
{:else if firstRun}
	<EmptyState
		title={firstRunTitle}
		description={firstRunDescription}
		icon={firstRunIcon}
		action={firstRunAction}
	/>
{:else}
	<div class="stats">
		{#each stats as stat (stat.label)}
			<StatTile {...stat} />
		{/each}
	</div>
	{#if children}<div class="sections">{@render children()}</div>{/if}
{/if}

<style>
	.stats {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
		gap: var(--space-4);
		margin: 0 0 var(--space-8);
	}
	.sections {
		display: grid;
		gap: var(--space-6);
	}
	.sections :global(h2) {
		font-size: var(--text-heading);
	}
</style>
