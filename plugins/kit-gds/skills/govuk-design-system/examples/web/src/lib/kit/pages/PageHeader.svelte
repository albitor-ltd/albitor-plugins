<!-- The top of every page. The one <h1> lives here, with an optional back link, a lead
     description and the page's actions. AppShell already supplies the main landmark, so a page
     never adds another. -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		description,
		back,
		actions
	}: {
		title: string;
		description?: string;
		/** A link to the parent page, shown above the title: `{ href, label }`. */
		back?: { href: string; label: string };
		actions?: Snippet;
	} = $props();
</script>

<header>
	{#if back}<a class="govuk-back-link" href={back.href}>{back.label}</a>{/if}
	<h1 class="govuk-heading-l">{title}</h1>
	{#if description}<p class="govuk-body-l">{description}</p>{/if}
	{#if actions}<div class="govuk-button-group">{@render actions()}</div>{/if}
</header>
