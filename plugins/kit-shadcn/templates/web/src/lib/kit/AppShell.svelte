<!-- Seeded by Albitor (kit-shadcn), then yours: the page frame. web-svelte's root layout renders it
     with the merged nav and the current path. Keep the skip link first and one main landmark. -->
<script lang="ts">
	import './tokens.css';
	import type { Snippet } from 'svelte';
	import type { NavGroup } from '../shell/types';
	import Toaster from './Toaster.svelte';

	let {
		nav,
		path,
		title = 'App',
		children
	}: { nav: NavGroup[]; path: string; title?: string; children: Snippet } = $props();

	function normal(p: string): string {
		return p.length > 1 ? p.replace(/\/+$/, '') : p;
	}

	/** 'page' on the page itself, 'true' inside its section, undefined elsewhere. */
	function current(href: string): 'page' | 'true' | undefined {
		const here = normal(path);
		const target = normal(href);
		if (here === target) return 'page';
		if (target !== '/' && here.startsWith(target + '/')) return 'true';
		return undefined;
	}
</script>

<a class="skip" href="#main">Skip to main content</a>
<div class="frame">
	<header>
		<span class="brand">{title}</span>
		<nav aria-label="Main">
			{#each nav as group (group.id ?? '')}
				<ul>
					{#each group.items as item (item.href)}
						<li>
							<a href={item.href} aria-current={current(item.href)}>{item.label}</a>
						</li>
					{/each}
				</ul>
			{/each}
		</nav>
	</header>
	<main id="main" tabindex="-1">
		{@render children()}
	</main>
</div>
<Toaster />

<style>
	.skip {
		position: absolute;
		left: 0.5rem;
		top: -4rem;
		padding: 0.5rem 1rem;
		background: var(--background);
		color: var(--foreground);
		border: 2px solid var(--ring);
		border-radius: var(--radius);
		z-index: 10;
	}
	.skip:focus {
		top: 0.5rem;
	}
	.frame {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
	}
	header {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.5rem 2rem;
		padding: 0.75rem 1.5rem;
		border-bottom: 1px solid var(--border);
	}
	.brand {
		font-weight: 600;
		font-size: var(--text-heading);
	}
	nav {
		display: flex;
		flex-wrap: wrap;
		gap: 0.25rem 1.5rem;
	}
	ul {
		display: flex;
		flex-wrap: wrap;
		gap: 0.25rem;
		margin: 0;
		padding: 0;
		list-style: none;
	}
	nav a {
		display: inline-block;
		padding: 0.5rem 0.75rem;
		border-radius: var(--radius);
		color: var(--foreground);
		text-decoration: none;
	}
	nav a:hover {
		background: var(--muted);
	}
	nav a[aria-current] {
		font-weight: 600;
		text-decoration: underline;
		text-underline-offset: 0.25rem;
	}
	a:focus-visible,
	main:focus-visible {
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}
	main {
		flex: 1;
		padding: 1.5rem;
		max-width: 72rem;
		width: 100%;
		box-sizing: border-box;
	}
</style>
