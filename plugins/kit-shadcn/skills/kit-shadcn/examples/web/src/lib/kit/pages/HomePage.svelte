<!-- A public home page: what this is, who it is for and the one next step. A headline, one plain
     sentence, a primary action (and at most one quieter link), then a few reasons in Cards and a
     closing prompt. No imagery: the look's typography and spacing do the work, and an icon sits
     only beside a reason's heading. The one <h1> is the headline.

       <HomePage title="Book a pitch at the club" lead="See what is free and reserve in a minute."
                 primary={{ href: '/pitches', label: 'See free pitches' }}
                 highlights={[{ title: 'Live availability', text: 'Slots update as they are booked.' }]} />

     `children` takes extra sections (steps, prices, a quote), each with its own <h2>. -->
<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import Card from '../Card.svelte';
	import Icon from '../Icon.svelte';
	import ActionLink from './ActionLink.svelte';

	type Glyph = Component<{ size?: number | string; class?: string }>;

	let {
		title,
		lead,
		primary,
		secondary,
		highlightsTitle = 'Why people use it',
		highlights = [],
		closing,
		children
	}: {
		/** The headline: what the service does for the person, in a few words. */
		title: string;
		/** One or two plain sentences under the headline. */
		lead: string;
		/** The one next step. */
		primary: { href: string; label: string };
		/** A quieter second route, such as "How it works". */
		secondary?: { href: string; label: string };
		highlightsTitle?: string;
		highlights?: Array<{ title: string; text: string; icon?: Glyph }>;
		/** A last prompt below the sections. */
		closing?: { title: string; text: string; action: { href: string; label: string } };
		children?: Snippet;
	} = $props();
</script>

<section class="hero">
	<h1>{title}</h1>
	<p class="lead">{lead}</p>
	<div class="actions">
		<ActionLink href={primary.href}>{primary.label}</ActionLink>
		{#if secondary}<ActionLink href={secondary.href} variant="secondary">{secondary.label}</ActionLink>{/if}
	</div>
</section>

{#if highlights.length > 0}
	<section class="highlights">
		<h2>{highlightsTitle}</h2>
		<ul>
			{#each highlights as item (item.title)}
				<li>
					<Card>
						<h3>
							{#if item.icon}<Icon icon={item.icon} size={20} />{/if}
							{item.title}
						</h3>
						<p>{item.text}</p>
					</Card>
				</li>
			{/each}
		</ul>
	</section>
{/if}

{#if children}<div class="more">{@render children()}</div>{/if}

{#if closing}
	<section class="closing">
		<Card tone="second">
			<h2>{closing.title}</h2>
			<p>{closing.text}</p>
			<ActionLink href={closing.action.href}>{closing.action.label}</ActionLink>
		</Card>
	</section>
{/if}

<style>
	.hero {
		max-width: 44rem;
		padding: var(--space-8) 0;
	}
	h1 {
		margin: 0 0 var(--space-3);
		font-size: clamp(var(--text-title), 5vw, 2.75rem);
		line-height: 1.15;
	}
	.lead {
		margin: 0 0 var(--space-6);
		font-size: calc(var(--text-body) * 1.15);
		color: var(--muted-foreground);
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
	}
	h2 {
		margin: 0 0 var(--space-4);
		font-size: var(--text-heading);
	}
	.highlights {
		margin-top: var(--space-6);
	}
	ul {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(15rem, 1fr));
		gap: var(--space-4);
		margin: 0;
		padding: 0;
		list-style: none;
	}
	h3 {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		margin: 0 0 var(--space-2);
		font-size: var(--text-body);
	}
	li p {
		margin: 0;
		color: var(--muted-foreground);
	}
	.more,
	.closing {
		margin-top: var(--space-8);
	}
	.closing p {
		margin: 0 0 var(--space-4);
	}
</style>
