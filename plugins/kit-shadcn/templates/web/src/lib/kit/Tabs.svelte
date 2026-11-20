<!-- A starting point, then yours: views of one thing, on Bits UI's Tabs (arrow keys move between
     tabs, Home and End jump, the panel is tied to its tab). Use it for peer views of the same
     data, not for navigation between pages: that is a link in the nav.

       <Tabs label="Report sections" tabs={[{ value: 'a', label: 'Summary' }, { value: 'b', label: 'Detail' }]}>
         {#snippet panel(value)}{#if value === 'a'}...{:else}...{/if}{/snippet}
       </Tabs> -->
<script lang="ts">
	import { Tabs } from 'bits-ui';
	import type { Snippet } from 'svelte';

	let {
		label,
		tabs,
		value = $bindable(),
		panel
	}: {
		/** Names the tab list. */
		label: string;
		tabs: Array<{ value: string; label: string }>;
		/** The selected tab; the first when not set. */
		value?: string;
		/** Renders one tab's panel. */
		panel: Snippet<[string]>;
	} = $props();

	// svelte-ignore state_referenced_locally
	value ??= tabs[0]?.value;
</script>

<Tabs.Root bind:value class="kit-tabs" data-kit="tabs">
	<Tabs.List aria-label={label} class="kit-tabs-list">
		{#each tabs as tab (tab.value)}
			<Tabs.Trigger value={tab.value} class="kit-tabs-trigger">{tab.label}</Tabs.Trigger>
		{/each}
	</Tabs.List>
	{#each tabs as tab (tab.value)}
		<Tabs.Content value={tab.value} class="kit-tabs-panel">{@render panel(tab.value)}</Tabs.Content>
	{/each}
</Tabs.Root>

<style>
	/* Bits UI renders the parts itself, so these are global by necessity. */
	:global(.kit-tabs-list) {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		border-bottom: var(--border-width) solid var(--border);
	}
	/* The hit area reaches 44px where the look draws a smaller tab. */
	:global(.kit-tabs-trigger::after) {
		content: '';
		position: absolute;
		inset-block: var(--hit-inset);
		inset-inline: 0;
	}
	:global(.kit-tabs-trigger) {
		position: relative;
		min-height: var(--control-height);
		padding: 0 var(--space-4);
		font: inherit;
		color: var(--muted-foreground);
		background: transparent;
		border: 0;
		border-bottom: calc(var(--border-width) * 2) solid transparent;
		margin-bottom: calc(var(--border-width) * -1);
		cursor: pointer;
		transition: color var(--motion-duration) var(--motion-ease);
	}
	:global(.kit-tabs-trigger:hover) {
		color: var(--foreground);
	}
	:global(.kit-tabs-trigger[data-state='active']) {
		color: var(--foreground);
		font-weight: var(--heading-weight);
		border-bottom-color: var(--primary);
	}
	:global(.kit-tabs-trigger:focus-visible),
	:global(.kit-tabs-panel:focus-visible) {
		outline: var(--focus-width) solid var(--ring);
		outline-offset: 2px;
	}
	:global(.kit-tabs-panel) {
		padding-top: var(--space-4);
	}
</style>
