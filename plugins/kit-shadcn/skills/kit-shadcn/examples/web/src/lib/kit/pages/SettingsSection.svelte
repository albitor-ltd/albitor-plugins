<!-- One titled group inside SettingsPage: a Card named by its heading, so a screen reader can jump
     between groups. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Card from '../Card.svelte';

	let {
		title,
		description,
		children
	}: { title: string; description?: string; children: Snippet } = $props();
</script>

<div class="section">
	<Card {title}>
		{#if description}<p class="description">{description}</p>{/if}
		{@render children()}
	</Card>
</div>

<style>
	.section {
		margin-bottom: var(--space-6);
	}
	.description {
		margin: 0 0 var(--space-4);
		color: var(--muted-foreground);
	}
</style>
