<!-- Seeded by Albitor (kit-gds), then yours: the page footer (component-kit@2 `Footer`) in GOV.UK
     footer colours and link style. One <footer> (the contentinfo landmark) that is a sibling of
     <main>, never inside it. Pass it to AppShell's `footer` prop so it sits at the foot of the
     page. Links are plain destinations (privacy, accessibility statement, cookies, contact);
     `children` is free text such as a copyright line. Both are optional, and with neither the
     footer renders nothing.
     It is NOT GOV.UK's footer component: no Crown copyright, no Open Government Licence line, no
     crest. Those are for services on GOV.UK only. -->
<script lang="ts" module>
	/** One destination in the footer. Standard across component-kit@2. */
	export type FooterLink = { href: string; label: string };
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	let { links = [], children }: { links?: FooterLink[]; children?: Snippet } = $props();
</script>

{#if links.length > 0 || children}
	<footer class="kit-footer">
		<div class="govuk-width-container">
			{#if links.length > 0}
				<nav aria-label="Footer">
					<ul class="kit-footer__list">
						{#each links as link, i (i)}
							<li class="kit-footer__item">
								<a class="govuk-link" href={link.href}>{link.label}</a>
							</li>
						{/each}
					</ul>
				</nav>
			{/if}
			{#if children}<div class="kit-footer__text">{@render children()}</div>{/if}
		</div>
	</footer>
{/if}

<style>
	/* GOV.UK footer: #f3f3f3 band under a 1px #b1b4b6 rule, text #0b0c0c. */
	.kit-footer {
		padding: 25px 0 15px;
		border-top: 1px solid #b1b4b6;
		background: #f3f3f3;
		color: #0b0c0c;
		font-family: arial, sans-serif;
		font-size: 1rem;
		line-height: 1.25;
	}
	.kit-footer__list {
		margin: 0 0 15px;
		padding: 0;
		list-style: none;
	}
	.kit-footer__item {
		display: inline-block;
		margin: 0 20px 10px 0;
	}
	.kit-footer__text {
		margin-bottom: 10px;
	}
	@media (min-width: 40.0625em) {
		.kit-footer {
			font-size: 1.1875rem;
			line-height: 1.3157894737;
		}
	}
</style>
