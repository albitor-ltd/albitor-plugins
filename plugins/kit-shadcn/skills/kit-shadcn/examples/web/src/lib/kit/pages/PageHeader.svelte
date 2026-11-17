<!-- The top of every page. The one <h1> lives here, with
     an optional description and the page's actions. AppShell already supplies the main landmark,
     so a page never adds another. -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		description,
		back,
		badge,
		actions
	}: {
		title: string;
		description?: string;
		/** A link to the parent page, shown above the title: `{ href, label }`. */
		back?: { href: string; label: string };
		/** A Pill beside the title, for the record's status. */
		badge?: Snippet;
		actions?: Snippet;
	} = $props();
</script>

<header>
	{#if back}<p class="back"><a href={back.href}>{back.label}</a></p>{/if}
	<div class="row">
		<div>
			<div class="title">
				<h1>{title}</h1>
				{#if badge}{@render badge()}{/if}
			</div>
			{#if description}<p class="description">{description}</p>{/if}
		</div>
		{#if actions}<div class="actions">{@render actions()}</div>{/if}
	</div>
</header>

<style>
	header {
		margin-bottom: var(--space-6);
	}
	.title {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-3);
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--space-4);
	}
	h1 {
		margin: 0;
		font-size: var(--text-title);
	}
	.description {
		margin: var(--space-1) 0 0;
		color: var(--muted-foreground);
	}
	.back {
		margin: 0 0 var(--space-2);
	}
	.back a {
		color: var(--link);
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}
</style>
