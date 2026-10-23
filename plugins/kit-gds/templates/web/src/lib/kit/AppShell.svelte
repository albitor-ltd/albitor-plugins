<!-- Seeded by Albitor (kit-gds), then yours: the page frame in GOV.UK Frontend v6 markup. web-svelte's
     root layout renders it with the merged nav and the current path. Keep the skip link first, one
     labelled nav and one main landmark.
     The header is TEXT ONLY: no Crown, no GOV.UK logotype (they are for services on GOV.UK only). -->
<script lang="ts">
	import './govuk.scss';
	import { onDestroy } from 'svelte';
	import type { Snippet } from 'svelte';
	import type { NavGroup } from '../shell/types';

	let {
		nav,
		path,
		title = 'App',
		children
	}: { nav: NavGroup[]; path: string; title?: string; children: Snippet } = $props();

	// GOV.UK Frontend's page template runs an inline script as the first thing in <body> that adds
	// `js-enabled` (and `govuk-frontend-supported` when the browser passes its feature check). This
	// app is a client-rendered SPA, so the same check runs here instead: no inline script, so no CSP
	// exception. It runs in the script body, not onMount, because Svelte mounts children first and
	// a child calling initAll() or new Button() in its own onMount must already see the classes.
	// The classes only gate JS-enhanced styling; without JS the page is the plain one. Only what
	// this added is removed on destroy.
	const added: string[] = [];
	if (typeof document !== 'undefined') {
		const wanted = ['js-enabled'];
		if ('noModule' in HTMLScriptElement.prototype) wanted.push('govuk-frontend-supported');
		for (const c of wanted) {
			if (!document.body.classList.contains(c)) {
				document.body.classList.add(c);
				added.push(c);
			}
		}
	}
	onDestroy(() => {
		if (typeof document !== 'undefined') document.body.classList.remove(...added);
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

<a href="#main-content" class="govuk-skip-link" data-module="govuk-skip-link">Skip to main content</a>
<header class="kit-header">
	<div class="govuk-width-container">
		<span class="kit-header__name">{title}</span>
	</div>
</header>
<div class="govuk-service-navigation">
	<div class="govuk-width-container">
		<div class="govuk-service-navigation__container">
			<nav aria-label="Main" class="govuk-service-navigation__wrapper">
				{#each nav as group (group.id ?? '')}
					<ul class="govuk-service-navigation__list">
						{#each group.items as item (item.href)}
							{@const here = current(item.href)}
							<li
								class="govuk-service-navigation__item"
								class:govuk-service-navigation__item--active={here}
							>
								<a class="govuk-service-navigation__link" href={item.href} aria-current={here}>
									{#if here}<strong class="govuk-service-navigation__active-fallback"
											>{item.label}</strong
										>{:else}{item.label}{/if}
								</a>
							</li>
						{/each}
					</ul>
				{/each}
			</nav>
		</div>
	</div>
</div>
<div class="govuk-width-container">
	<main class="govuk-main-wrapper" id="main-content" tabindex="-1">
		{@render children()}
	</main>
</div>

<style>
	/* A text-only band in the brand colour (`$govuk-brand-colour` in govuk.scss, the GOV.UK blue
	   unless the app is off GOV.UK and set its own): GOV.UK's header shape, none of its branding.
	   The text colour is chosen in govuk.scss to keep AA contrast on that colour. */
	.kit-header {
		background: var(--govuk-brand-colour, #1d70b8);
		color: var(--kit-header-text, #fff);
		border-bottom: 10px solid #0b0c0c;
		padding: 15px 0 10px;
	}
	.kit-header__name {
		font-family: arial, sans-serif;
		font-size: 1.5rem;
		font-weight: 700;
		line-height: 1.1;
	}
	/* Only from tablet up, where the items sit in a row: on mobile the stylesheet's bottom margin
	   spaces the stacked lists. */
	@media (min-width: 40.0625em) {
		:global(.govuk-service-navigation__list) {
			margin-block: 0;
		}
	}
	:global(main:focus) {
		outline: none;
	}
</style>
