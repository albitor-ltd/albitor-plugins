<!-- Seeded by Albitor (kit-shadcn), then yours: the page footer (component-kit@2 `Footer`). One
     <footer> (the contentinfo landmark) that is a sibling of <main>, never inside it. Pass it to
     AppShell's `footer` prop so it sits at the foot of the frame. Links are plain destinations
     (privacy, terms, accessibility statement, contact); `children` is free text such as a copyright
     line. Both are optional, and with neither the footer renders nothing. -->
<script lang="ts" module>
	/** One destination in the footer. Standard across component-kit@2. */
	export type FooterLink = { href: string; label: string };
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	let { links = [], children }: { links?: FooterLink[]; children?: Snippet } = $props();
</script>

{#if links.length > 0 || children}
	<footer>
		{#if links.length > 0}
			<nav aria-label="Footer">
				<ul>
					{#each links as link, i (i)}
						<li><a href={link.href}>{link.label}</a></li>
					{/each}
				</ul>
			</nav>
		{/if}
		{#if children}<div class="text">{@render children()}</div>{/if}
	</footer>
{/if}

<style>
	footer {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem 2rem;
		padding: 1rem 1.5rem;
		border-top: 1px solid var(--border);
		color: var(--muted-foreground);
		font-size: var(--text-small);
	}
	ul {
		display: flex;
		flex-wrap: wrap;
		gap: 0.25rem 1.5rem;
		margin: 0;
		padding: 0;
		list-style: none;
	}
	a {
		color: var(--link);
	}
	a:focus-visible {
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}
</style>
