<!-- Seeded by Albitor (kit-gds), then yours: the page frame in GOV.UK Frontend v6 markup. web-svelte's
     root layout renders it with the merged nav and the current path. Keep the skip link first, one
     labelled nav and one main landmark.
     The header is TEXT ONLY: no Crown, no GOV.UK logotype (they are for services on GOV.UK only). -->
<script lang="ts">
	import './govuk.scss';
	import type { Snippet } from 'svelte';
	import type { NavGroup } from '../shell/types';

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
	/* A plain black band carrying the service name as text: GOV.UK's header colours, none of its
	   branding. White on #0b0c0c is 19.3:1. */
	.kit-header {
		background: #0b0c0c;
		color: #fff;
		border-bottom: 10px solid #1d70b8;
		padding: 15px 0 10px;
	}
	.kit-header__name {
		font-family: arial, sans-serif;
		font-size: 1.5rem;
		font-weight: 700;
		line-height: 1.1;
	}
	:global(.govuk-service-navigation__list) {
		margin-block: 0;
	}
	:global(main:focus) {
		outline: none;
	}
</style>
