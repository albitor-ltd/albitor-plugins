<!-- Seeded by Albitor (kit-carbon), then yours: one titled group inside SettingsPage. The section is
     named by its heading, so a screen reader can jump between groups. -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		description,
		children
	}: { title: string; description?: string; children: Snippet } = $props();

	const uid = $props.id();
</script>

<section aria-labelledby={`${uid}-title`}>
	<h2 id={`${uid}-title`}>{title}</h2>
	{#if description}<p class="description">{description}</p>{/if}
	{@render children()}
</section>

<style>
	section {
		margin-bottom: 2rem;
		padding-bottom: 1rem;
		border-bottom: 1px solid var(--cds-border-strong);
	}
	h2 {
		margin: 0 0 0.25rem;
		font-size: 1.25rem;
	}
	.description {
		margin: 0 0 1rem;
		color: var(--cds-text-secondary);
	}
</style>
