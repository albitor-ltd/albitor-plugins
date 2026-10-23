<!-- Seeded by Albitor (kit-carbon), then yours: the page footer (component-kit@2 `Footer`), in
     Carbon's tokens (Carbon has no footer component). One <footer> (the contentinfo landmark) that
     is a sibling of <main>, never inside it. Pass it to AppShell's `footer` prop so it sits at the
     foot of the frame, clear of the side nav. Links are plain destinations (privacy, terms,
     accessibility statement, contact); `children` is free text such as a copyright line. Both are
     optional, and with neither the footer renders nothing. -->
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
		padding: 1rem 2rem;
		border-top: 1px solid var(--cds-border-subtle);
		color: var(--cds-text-secondary);
		font-size: 0.875rem;
		line-height: 1.28572;
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
		color: var(--cds-link-primary);
	}
	a:hover {
		color: var(--cds-link-primary-hover);
	}
	a:focus-visible {
		outline: 2px solid var(--cds-focus);
		outline-offset: 2px;
	}
</style>
