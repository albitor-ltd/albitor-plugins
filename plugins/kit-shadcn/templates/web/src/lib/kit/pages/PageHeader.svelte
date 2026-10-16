<!-- Seeded by Albitor (kit-shadcn), then yours: the top of every page. The one <h1> lives here, with
     an optional description and the page's actions. AppShell already supplies the main landmark,
     so a page never adds another. -->
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
	{#if back}<p class="back"><a href={back.href}>{back.label}</a></p>{/if}
	<div class="row">
		<div>
			<h1>{title}</h1>
			{#if description}<p class="description">{description}</p>{/if}
		</div>
		{#if actions}<div class="actions">{@render actions()}</div>{/if}
	</div>
</header>

<style>
	header {
		margin-bottom: 1.5rem;
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
	}
	h1 {
		margin: 0;
		font-size: var(--text-title);
	}
	.description {
		margin: 0.25rem 0 0;
		color: var(--muted-foreground);
	}
	.back {
		margin: 0 0 0.5rem;
	}
	.back a {
		color: var(--link);
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
</style>
