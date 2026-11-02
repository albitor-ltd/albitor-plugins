<!-- One titled group inside SettingsPage. The section is named by its heading, so a screen reader
     can jump between groups. -->
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
	<h2 class="govuk-heading-m" id={`${uid}-title`}>{title}</h2>
	{#if description}<p class="govuk-body">{description}</p>{/if}
	{@render children()}
</section>
