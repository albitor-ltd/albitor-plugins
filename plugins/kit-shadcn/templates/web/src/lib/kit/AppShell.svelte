<!-- Seeded by Albitor (kit-shadcn), then yours: the page frame (component-kit@2 `AppShell`).
     web-svelte's root layout renders it with the merged nav and the current path. Keep the skip
     link first and one main landmark. `footer` is an optional snippet rendered after <main>,
     outside it, normally `<Footer>`. `account` is an optional snippet for the signed-in visitor's
     controls (Sign out, a link to their settings), shown at the right of the header on every
     width. Below 48rem the nav collapses behind a Menu button (a disclosure, not a dialog). -->
<script lang="ts">
	import './tokens.css';
	import type { Snippet } from 'svelte';
	import type { NavGroup } from '../shell/types';
	import Toaster from './Toaster.svelte';

	let {
		nav,
		path,
		title = 'App',
		footer,
		account,
		children
	}: {
		nav: NavGroup[];
		path: string;
		title?: string;
		footer?: Snippet;
		account?: Snippet;
		children: Snippet;
	} = $props();

	// The phone menu. Closed on every navigation and on Escape, so the page is never left hidden
	// behind it. Above 48rem the button is not shown and the nav is always visible.
	let menuOpen = $state(false);
	const hasNav = $derived(nav.some((g) => g.items.length > 0));
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
	<header onkeydown={closeOnEscape}>
		<span class="brand">{title}</span>
		{#if hasNav}
			<button
				type="button"
				class="menu"
				bind:this={menuButton}
				aria-expanded={menuOpen}
				aria-controls="main-nav"
				onclick={() => (menuOpen = !menuOpen)}
			>
				Menu
			</button>
		{/if}
		<nav
			id="main-nav"
			aria-label="Main"
			class:open={menuOpen}
		>
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
		{#if account}<div class="account">{@render account()}</div>{/if}
	</header>
	<main id="main" tabindex="-1">
		{@render children()}
	</main>
	{@render footer?.()}
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
	.account {
		margin-inline-start: auto;
	}
	.menu {
		display: none;
		min-height: 2.75rem;
		padding: 0.5rem 1rem;
		font: inherit;
		color: var(--foreground);
		background: transparent;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		cursor: pointer;
	}
	/* Phone: one header row (name, account, Menu); the nav opens under it as one full-width
	   list of 44px-tall links. Nothing wraps into a second row of tabs. */
	@media (width < 48rem) {
		header {
			gap: 0.5rem 0.75rem;
			padding: 0.5rem 1rem;
		}
		.menu {
			display: inline-block;
			margin-inline-start: auto;
		}
		.account {
			margin-inline-start: 0;
		}
		nav {
			display: none;
			order: 1;
			flex-basis: 100%;
			flex-direction: column;
			flex-wrap: nowrap;
			gap: 0;
		}
		nav.open {
			display: flex;
		}
		ul {
			flex-direction: column;
			flex-wrap: nowrap;
			gap: 0;
		}
		nav a {
			display: block;
			min-height: 2.75rem;
			box-sizing: border-box;
			padding: 0.75rem;
		}
		main {
			padding: 1rem;
		}
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
	.menu:focus-visible,
	main:focus-visible {
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}
	main {
		flex: 1;
		padding: 1.5rem;
		max-width: 72rem;
		margin-inline: auto;
		width: 100%;
		box-sizing: border-box;
	}
</style>
