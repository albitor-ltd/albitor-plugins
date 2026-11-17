<!-- A starting point, then yours: the GOV.UK service navigation (Frontend 6.4.0 markup): the service
     name and the links between its top-level pages, as its own band. AppShell already renders one
     from the app's nav, so use this only for a band of your own (a second level, or a page with no
     shell). The current page is marked (aria-current, the active bar). There is no menu toggle: the
     links wrap on a phone. Keep the links to a handful. -->
<script module lang="ts">
	export interface ServiceNavItem {
		label: string;
		href: string;
	}
</script>

<script lang="ts">
	let {
		serviceName,
		serviceHref = '/',
		items,
		path,
		label = 'Menu'
	}: {
		serviceName?: string;
		serviceHref?: string;
		items: ServiceNavItem[];
		/** The current path, to mark the current page. */
		path: string;
		/** The accessible name of the links' navigation landmark. */
		label?: string;
	} = $props();

	function normal(p: string): string {
		return p.length > 1 && p.endsWith('/') ? p.slice(0, -1) : p;
	}

	function current(href: string): 'page' | 'true' | undefined {
		const here = normal(path);
		const target = normal(href);
		if (here === target) return 'page';
		if (target !== '/' && here.startsWith(target + '/')) return 'true';
		return undefined;
	}
</script>

<section aria-label="Service information" class="govuk-service-navigation">
	<div class="govuk-width-container">
		<div class="govuk-service-navigation__container">
			{#if serviceName}
				<span class="govuk-service-navigation__service-name">
					<a class="govuk-service-navigation__link" href={serviceHref}>{serviceName}</a>
				</span>
			{/if}
			{#if items.length}
				<nav aria-label={label} class="govuk-service-navigation__wrapper">
					<ul class="govuk-service-navigation__list">
						{#each items as item (item.href)}
							{@const here = current(item.href)}
							<li class={['govuk-service-navigation__item', here && 'govuk-service-navigation__item--active']}>
								<a class="govuk-service-navigation__link" href={item.href} aria-current={here}>
									{#if here}<strong class="govuk-service-navigation__active-fallback">{item.label}</strong
										>{:else}{item.label}{/if}
								</a>
							</li>
						{/each}
					</ul>
				</nav>
			{/if}
		</div>
	</div>
</section>
