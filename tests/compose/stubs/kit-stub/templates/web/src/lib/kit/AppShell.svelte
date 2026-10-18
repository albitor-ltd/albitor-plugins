<!-- The smallest AppShell that satisfies the web-svelte component-kit contract. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { NavGroup } from '../shell/types';

	let {
		nav,
		path,
		title,
		children
	}: { nav: NavGroup[]; path: string; title: string; children: Snippet } = $props();
</script>

<a href="#main">Skip to content</a>
<header>{title}</header>
<nav aria-label="Main">
	{#each nav as group (group.id ?? '')}
		<ul>
			{#each group.items as item (item.href)}
				<li><a href={item.href} aria-current={path === item.href ? 'page' : undefined}>{item.label}</a></li>
			{/each}
		</ul>
	{/each}
</nav>
<main id="main">
	{@render children()}
</main>
