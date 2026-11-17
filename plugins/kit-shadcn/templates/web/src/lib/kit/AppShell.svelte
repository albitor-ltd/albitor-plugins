<!-- A starting point, then yours: the page frame.
     web-svelte's root layout renders it with the merged nav and the current path. Keep the skip
     link first and one main landmark. `footer` is an optional snippet rendered after <main>,
     outside it, normally `<Footer>`. `account` is an optional snippet for the signed-in visitor's
     controls (Sign out, a link to their settings), shown at the right of the header on every
     width. `mark` is the optional URL of the app's logo, drawn beside the name with an empty alt
     (web-svelte's layout passes `/brand/mark.svg`; do not redraw it).
     The navigation is NavShell's. `layout` puts it in the header row (`top`, the default) or in a
     column beside the page (`side`). `phoneNav` is the look's stance for a phone: `drawer` (the
     default) collapses it behind a Menu button (a disclosure, not a dialog); `tabs` makes it a
     bottom tab bar (NavShell.svelte has the 2 to 5, 4 plus More and 1 destination rules). `icons`
     maps a destination's href to a Lucide icon. -->
<script lang="ts">
	import './tokens.css';
	import type { Component, Snippet } from 'svelte';
	import Menu from '@lucide/svelte/icons/menu';
	import type { NavGroup } from '../shell/types';
	import Icon from './Icon.svelte';
	import NavShell, { phoneMode, type PhoneNav } from './NavShell.svelte';
	import Toaster from './Toaster.svelte';

	let {
		nav,
		path,
		title = 'App',
		mark,
		footer,
		account,
		layout = 'top',
		phoneNav = 'drawer',
		icons,
		children
	}: {
		nav: NavGroup[];
		path: string;
		title?: string;
		/** URL of the app's mark (the platform-written `/brand/mark.svg`). Decorative: the name sits beside it. */
		mark?: string;
		footer?: Snippet;
		account?: Snippet;
		/** Where the navigation sits on a wide screen. */
		layout?: 'top' | 'side';
		/** What the phone gets: the look's stance. */
		phoneNav?: PhoneNav;
		/** href to Lucide icon, drawn beside each destination. */
		icons?: Record<string, Component<{ size?: number | string; class?: string }>>;
		children: Snippet;
	} = $props();

	// The mark is decorative (the name is beside it) and hidden when its file is missing, so an app
	// with no platform-written brand files shows the name alone.
	let markFailed = $state(false);

	// The phone menu. Closed on every navigation and on Escape, so the page is never left hidden
	// behind it. Above 48rem the button is not shown and the nav is always visible.
	let menuOpen = $state(false);
	const hasNav = $derived(nav.some((g) => g.items.length > 0));
	const phone = $derived(phoneMode(nav, phoneNav));
	$effect(() => {
		void path;
		menuOpen = false;
	});

	let menuButton: HTMLButtonElement | undefined = $state();
	function closeOnEscape(e: KeyboardEvent) {
		if (e.key === 'Escape' && menuOpen) {
			menuOpen = false;
			menuButton?.focus();
		}
	}
</script>

<a class="skip" href="#main">Skip to main content</a>
<div class="frame" data-phone={phone} data-layout={layout}>
	<header onkeydown={closeOnEscape}>
		<span class="brand">
			{#if mark && !markFailed}<img class="mark" src={mark} alt="" onerror={() => (markFailed = true)} />{/if}
			{title}
		</span>
		{#if hasNav && phone === 'drawer'}
			<button
				type="button"
				class="menu"
				bind:this={menuButton}
				aria-expanded={menuOpen}
				aria-controls="main-nav"
				onclick={() => (menuOpen = !menuOpen)}
			>
				<Icon icon={Menu} size={18} />
				Menu
			</button>
		{/if}
		{#if layout === 'top'}
			<NavShell {nav} {path} {phoneNav} {icons} open={menuOpen} />
		{/if}
		{#if account}<div class="account">{@render account()}</div>{/if}
	</header>
	<div class="body">
		{#if layout === 'side'}
			<aside><NavShell {nav} {path} {phoneNav} {icons} open={menuOpen} side /></aside>
		{/if}
		<main id="main" tabindex="-1">
			{@render children()}
		</main>
	</div>
	{@render footer?.()}
	<Toaster />
</div>

<style>
	.skip {
		position: absolute;
		left: var(--space-2);
		top: -4rem;
		padding: var(--space-2) var(--space-4);
		background: var(--background);
		color: var(--foreground);
		border: var(--focus-width) solid var(--ring);
		border-radius: var(--radius);
		z-index: 10;
	}
	.skip:focus {
		top: var(--space-2);
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
		gap: var(--space-2) var(--space-8);
		padding: var(--space-3) var(--space-6);
		background: var(--header);
		color: var(--header-ink);
		border-bottom: var(--border-width) solid var(--border);
		box-shadow: var(--shadow);
	}
	.brand {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		font-family: var(--heading-font);
		font-weight: var(--heading-weight);
		letter-spacing: var(--heading-tracking);
		font-size: var(--text-heading);
	}
	.mark {
		height: 1.75rem;
		width: auto;
		max-width: 3rem;
		object-fit: contain;
	}
	.account {
		margin-inline-start: auto;
	}
	.menu {
		display: none;
		align-items: center;
		gap: var(--space-2);
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-4);
		font: inherit;
		color: var(--header-ink);
		background: transparent;
		border: var(--border-width) solid var(--border);
		border-radius: var(--radius);
		cursor: pointer;
	}
	.body {
		flex: 1;
		display: flex;
		flex-direction: column;
	}
	/* Phone: one header row (name, account, Menu); a drawer's list opens under it. */
	@media (width < 48rem) {
		header {
			gap: var(--space-2) var(--space-3);
			padding: var(--space-2) var(--space-4);
		}
		.menu {
			display: inline-flex;
			margin-inline-start: auto;
		}
		.account {
			margin-inline-start: 0;
		}
		main {
			padding: var(--space-4);
		}
		/* Room under the page for the fixed tab bar and the home indicator. */
		.frame[data-phone='tabs'] {
			/* The bar's height follows the text size; the page and the toasts clear it. */
			--tab-bar-height: max(
				3.75rem,
				calc(var(--text-small) * 1.5 + 1.25rem + var(--space-2) * 2 + var(--border-width) * 2)
			);
			--tab-bar-offset: calc(var(--tab-bar-height) + env(safe-area-inset-bottom, 0px));
			padding-block-end: var(--tab-bar-offset);
		}
		aside {
			display: contents;
		}
	}
	@media (width >= 48rem) {
		.frame[data-layout='side'] .body {
			display: grid;
			grid-template-columns: 14rem minmax(0, 1fr);
			align-items: start;
		}
		aside {
			padding: var(--space-4) var(--space-3);
			border-inline-end: var(--border-width) solid var(--border);
			align-self: stretch;
		}
	}
	.skip:focus-visible,
	.menu:focus-visible,
	main:focus-visible {
		outline: var(--focus-width) solid var(--ring);
		outline-offset: 2px;
	}
	main {
		flex: 1;
		padding: var(--space-6);
		max-width: 72rem;
		margin-inline: auto;
		width: 100%;
		box-sizing: border-box;
	}
</style>
