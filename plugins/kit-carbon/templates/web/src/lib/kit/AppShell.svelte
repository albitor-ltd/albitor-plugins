<!-- Seeded by Albitor (kit-carbon), then yours: the page frame, Carbon's UI shell (header, side nav,
     content). web-svelte's root layout renders it with the merged nav and the current path. Keep
     the skip link first, one labelled nav and one main landmark. `footer` is an optional snippet
     rendered after <main>, outside it, normally `<Footer>`. -->
<script lang="ts">
	import 'carbon-components-svelte/css/all.css';
	import {
		Content,
		Header,
		SideNav,
		SideNavDivider,
		SideNavItems,
		SideNavLink,
		SkipToContent,
		Theme
	} from 'carbon-components-svelte';
	import { onMount } from 'svelte';
	import type { Snippet } from 'svelte';
	import type { NavGroup } from '../shell/types';
	import { pickTheme, type CarbonTheme } from './theme';
	import Toaster from './Toaster.svelte';

	let {
		nav,
		path,
		title = 'App',
		footer,
		children
	}: {
		nav: NavGroup[];
		path: string;
		title?: string;
		footer?: Snippet;
		children: Snippet;
	} = $props();

	let theme: CarbonTheme = $state('white');
	let sideNavOpen = $state(true);

	// Follow the visitor's preference, and a data-theme the page sets (or sets later).
	onMount(() => {
		const query = window.matchMedia?.('(prefers-color-scheme: dark)');
		const apply = () =>
			(theme = pickTheme(document.documentElement.dataset.theme, query?.matches ?? false));
		apply();
		query?.addEventListener('change', apply);
		const watch = new MutationObserver(apply);
		watch.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });
		return () => {
			query?.removeEventListener('change', apply);
			watch.disconnect();
		};
	});

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

<Theme bind:theme />
<Header platformName={title} aria-label={title} bind:isSideNavOpen={sideNavOpen}>
	<svelte:fragment slot="skipToContent"><SkipToContent /></svelte:fragment>
</Header>
<SideNav bind:isOpen={sideNavOpen} ariaLabel="Main">
	<SideNavItems>
		{#each nav as group, g (group.id ?? '')}
			{#if g > 0}<SideNavDivider />{/if}
			{#each group.items as item (item.href)}
				{@const here = current(item.href)}
				<SideNavLink
					href={item.href}
					text={item.label}
					isSelected={here === 'page'}
					aria-current={here}
				/>
			{/each}
		{/each}
	</SideNavItems>
</SideNav>
<Content tabindex={-1}>
	{@render children()}
</Content>
{#if footer}
	<!-- Carbon's Content is always a <main>, so the footer is its sibling. It takes the same
	     left margin Carbon gives Content beside an expanded side nav (16rem), and none when the
	     nav is closed or overlays the page (below the lg breakpoint, 66rem). -->
	<div class="footer-offset">{@render footer()}</div>
{/if}
<Toaster />

<style>
	@media (min-width: 66rem) {
		:global(.bx--side-nav.bx--side-nav--expanded) ~ .footer-offset {
			margin-left: 16rem;
		}
	}
</style>
