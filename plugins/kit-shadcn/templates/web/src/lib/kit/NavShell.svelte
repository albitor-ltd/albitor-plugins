<!-- A starting point, then yours: the app's navigation, from the one destinations list AppShell
     receives. Structure and style are separate: this file decides where the destinations sit, the
     tokens decide how they look.

     Desktop: a row of links (the top bar), or a column when `side`. Phone, by `phoneNav`:
       'drawer'  a Menu button (in AppShell's header) opens the list under it;
       'tabs'    a fixed bottom tab bar for 2 to 5 destinations; with 6 or more, 4 tabs and a More
                 button that opens the rest as a list. One destination is not worth a bar, so it
                 falls back to 'drawer'.
     Pick the one the look leans to (its "Phone navigation" stance) and never both: the same
     destinations are not in a tab bar and a menu. Every destination stays reachable on phone.
     `aria-current="page"` is on the current link, never on More: when the current page is in More the
     button says so in words. Targets are 44px; the bar pads for the home indicator. `icons` maps an
     href to a Lucide icon, drawn beside the label (decorative). -->
<script module lang="ts">
	import type { NavGroup, NavItem } from '../shell/types';

	export type PhoneNav = 'drawer' | 'tabs';

	/** How many tabs a bar holds before the rest move behind More. */
	const BAR_MAX = 5;
	const BAR_WITH_MORE = 4;

	/** What the phone really gets: tabs only for 2 or more destinations, else the drawer. */
	export function phoneMode(nav: NavGroup[], want: PhoneNav): PhoneNav {
		return want === 'tabs' && nav.reduce((n, g) => n + g.items.length, 0) >= 2 ? 'tabs' : 'drawer';
	}

	function visibleItems(nav: NavGroup[]): NavItem[] {
		return nav.flatMap((g) => g.items);
	}

	/** The tabs, and what sits behind More (empty up to five). */
	export function splitTabs(items: NavItem[]): { tabs: NavItem[]; more: NavItem[] } {
		return items.length <= BAR_MAX
			? { tabs: items, more: [] }
			: { tabs: items.slice(0, BAR_WITH_MORE), more: items.slice(BAR_WITH_MORE) };
	}
</script>

<script lang="ts">
	import type { Component } from 'svelte';
	import Ellipsis from '@lucide/svelte/icons/ellipsis';
	import Icon from './Icon.svelte';

	let {
		nav,
		path,
		open = false,
		phoneNav = 'drawer',
		side = false,
		icons = {}
	}: {
		nav: NavGroup[];
		path: string;
		/** Drawer mode: whether the list is open on a phone. AppShell's Menu button owns it. */
		open?: boolean;
		phoneNav?: PhoneNav;
		/** Desktop: a column down the side instead of a row in the header. */
		side?: boolean;
		icons?: Record<string, Component<{ size?: number | string; class?: string }>>;
	} = $props();

	const mode = $derived(phoneMode(nav, phoneNav));
	const split = $derived(splitTabs(visibleItems(nav)));

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

	// More: closed on every navigation, on Escape (focus back on the button) and when focus leaves.
	let moreOpen = $state(false);
	let moreButton: HTMLButtonElement | undefined = $state();
	$effect(() => {
		void path;
		moreOpen = false;
	});
	const moreHasCurrent = $derived(split.more.some((i) => current(i.href)));

	function onkeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && moreOpen) {
			moreOpen = false;
			moreButton?.focus();
		}
	}
	function onfocusout(e: FocusEvent) {
		const next = e.relatedTarget as Node | null;
		if (moreOpen && next && !(e.currentTarget as HTMLElement).contains(next)) moreOpen = false;
	}
</script>

{#snippet link(item: NavItem)}
	<a href={item.href} aria-current={current(item.href)}>
		{#if icons[item.href]}<Icon icon={icons[item.href]} size={20} />{/if}
		<span>{item.label}</span>
	</a>
{/snippet}

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<nav
	id="main-nav"
	aria-label="Main"
	class:open
	class:side
	class:tabs={mode === 'tabs'}
	data-phone={mode}
	{onkeydown}
	{onfocusout}
>
	{#if mode === 'drawer'}
		{#each nav as group (group.id ?? '')}
			<ul>
				{#each group.items as item (item.href)}
					<li>{@render link(item)}</li>
				{/each}
			</ul>
		{/each}
	{:else}
		<ul class="items">
			{#each split.tabs as item (item.href)}
				<li>{@render link(item)}</li>
			{/each}
			{#if split.more.length}
				<li class="more-item">
					<button
						type="button"
						class="more-button"
						class:has-current={moreHasCurrent}
						bind:this={moreButton}
						aria-expanded={moreOpen}
						aria-controls="more-list"
						onclick={() => (moreOpen = !moreOpen)}
					>
						<Icon icon={Ellipsis} size={20} />
						<span>More</span>
						{#if moreHasCurrent}<span class="sr"> (current page is in More)</span>{/if}
					</button>
					<ul id="more-list" class="more" class:open={moreOpen}>
						{#each split.more as item (item.href)}
							<li>{@render link(item)}</li>
						{/each}
					</ul>
				</li>
			{/if}
		</ul>
	{/if}
</nav>

<style>
	nav {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1) var(--space-6);
	}
	ul {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		margin: 0;
		padding: 0;
		list-style: none;
	}
	a,
	.more-button {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		color: var(--header-ink);
		text-decoration: none;
		transition: background-color var(--motion-duration) var(--motion-ease);
	}
	a:hover,
	.more-button:hover {
		background: var(--muted);
	}
	a[aria-current] {
		font-weight: var(--heading-weight);
		text-decoration: underline;
		text-underline-offset: 0.25rem;
	}
	a:focus-visible,
	.more-button:focus-visible {
		outline: var(--focus-width) solid var(--ring);
		outline-offset: 2px;
	}
	.more-button {
		font: inherit;
		background: transparent;
		border: 0;
		cursor: pointer;
	}
	.sr {
		position: absolute;
		inline-size: 1px;
		block-size: 1px;
		margin: -1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}

	/* Desktop: More has nothing to hide, so its list flows inline and its button is not drawn. */
	@media (width >= 48rem) {
		.more-item,
		.more {
			display: contents;
		}
		.more-button {
			display: none;
		}
		.side {
			flex-direction: column;
			flex-wrap: nowrap;
		}
		.side ul,
		.side .items {
			flex-direction: column;
			flex-wrap: nowrap;
		}
		.side a {
			display: flex;
		}
		.side a[aria-current] {
			background: var(--second-subtle);
			color: var(--second-text);
			text-decoration: none;
		}
	}

	@media (width < 48rem) {
		/* Drawer: closed until the Menu button opens it, then one list of 44px links. */
		nav:not(.tabs) {
			display: none;
			order: 1;
			flex-basis: 100%;
			flex-direction: column;
			flex-wrap: nowrap;
			gap: 0;
		}
		nav.open:not(.tabs) {
			display: flex;
		}
		nav:not(.tabs) ul {
			flex-direction: column;
			flex-wrap: nowrap;
			gap: 0;
		}
		nav:not(.tabs) a {
			display: flex;
			min-height: 2.75rem;
			box-sizing: border-box;
			padding: var(--space-3);
		}

		/* Tabs: a bar fixed to the bottom edge, padded for the home indicator. */
		nav.tabs {
			position: fixed;
			inset-inline: 0;
			inset-block-end: 0;
			z-index: 5;
			display: block;
			padding-block-end: env(safe-area-inset-bottom, 0px);
			background: var(--header);
			border-top: var(--border-width) solid var(--border);
		}
		.items {
			display: grid;
			grid-auto-flow: column;
			grid-auto-columns: 1fr;
			flex-wrap: nowrap;
			gap: 0;
		}
		.tabs li {
			position: relative;
			min-inline-size: 0;
		}
		.tabs a,
		.tabs .more-button {
			display: flex;
			flex-direction: column;
			justify-content: center;
			gap: 2px;
			inline-size: 100%;
			min-height: 2.75rem;
			box-sizing: border-box;
			padding: var(--space-2) var(--space-1);
			border-radius: 0;
			font-size: var(--text-small);
			text-align: center;
			text-decoration: none;
		}
		.tabs a[aria-current='page'],
		.tabs .has-current {
			color: var(--foreground);
			font-weight: var(--heading-weight);
			box-shadow: inset 0 calc(var(--border-width) * 2) 0 var(--primary);
		}
		.more {
			display: none;
			position: absolute;
			inset-block-end: 100%;
			inset-inline-end: 0;
			min-inline-size: 12rem;
			flex-direction: column;
			flex-wrap: nowrap;
			gap: 0;
			background: var(--surface);
			border: var(--border-width) solid var(--border);
			border-radius: var(--radius);
			box-shadow: var(--shadow-overlay);
		}
		.more.open {
			display: flex;
		}
		.more a {
			flex-direction: row;
			justify-content: flex-start;
			text-align: start;
			padding: var(--space-3);
		}
	}
</style>
