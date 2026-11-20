<!-- An article: something to read. A header with a link back to the section, the headline, a
     standfirst and a byline, then the body at a comfortable reading width (about 65 characters a
     line, generous line height), and links to related reading. The body is the caller's markup:
     headings in order from <h2>, paragraphs, lists, a `<blockquote>`. Never `{@html}` of user
     input; render prose you wrote or sanitised.

       <ArticlePage title="Winter opening hours" description="What changes from 1 December."
                    back={{ href: '/news', label: 'All news' }} byline="Sam Rowe"
                    published="2026-11-20" related={[{ href: '/news/parking', label: 'Parking' }]}>
         <p>From 1 December the hall opens at 9am.</p>
         <h2>Weekends</h2>
         <p>Weekend hours stay the same.</p>
       </ArticlePage> -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import PageHeader from './PageHeader.svelte';
	import StateNotice from './StateNotice.svelte';

	let {
		title,
		description,
		back,
		byline,
		published,
		readingTime,
		related = [],
		relatedTitle = 'Keep reading',
		status = 'ready',
		errorMessage,
		onretry,
		children
	}: {
		title: string;
		/** The standfirst: one sentence on what the piece is about. */
		description?: string;
		back: { href: string; label: string };
		byline?: string;
		/** An ISO date (2026-11-20); shown as written, so format it before passing it if you prefer. */
		published?: string;
		/** For example "4 min read". */
		readingTime?: string;
		related?: Array<{ href: string; label: string }>;
		relatedTitle?: string;
		status?: 'loading' | 'error' | 'ready';
		errorMessage?: string;
		onretry?: () => void;
		children: Snippet;
	} = $props();

	const meta = $derived([byline, published, readingTime].filter(Boolean).join(' · '));
</script>

<PageHeader {title} {description} {back} />

{#if status !== 'ready'}
	<StateNotice {status} {errorMessage} {onretry} loadingLabel="Loading the article" />
{:else}
	<article>
		{#if meta}<p class="meta">{meta}</p>{/if}
		<div class="prose">{@render children()}</div>
	</article>
	{#if related.length > 0}
		<nav class="related" aria-labelledby="related-heading">
			<h2 id="related-heading">{relatedTitle}</h2>
			<ul>
				{#each related as link (link.href)}
					<li><a href={link.href}>{link.label}</a></li>
				{/each}
			</ul>
		</nav>
	{/if}
{/if}

<style>
	article,
	.related {
		max-width: 40rem;
	}
	.meta {
		margin: 0 0 var(--space-6);
		color: var(--muted-foreground);
		font-size: var(--text-small);
	}
	.prose {
		line-height: 1.65;
	}
	.prose :global(h2) {
		margin: var(--space-8) 0 var(--space-3);
		font-size: var(--text-heading);
	}
	.prose :global(h3) {
		margin: var(--space-6) 0 var(--space-2);
		font-size: var(--text-body);
	}
	.prose :global(p),
	.prose :global(ul),
	.prose :global(ol) {
		margin: 0 0 var(--space-4);
	}
	.prose :global(a) {
		color: var(--link);
	}
	.prose :global(blockquote) {
		margin: var(--space-6) 0;
		padding-inline-start: var(--space-4);
		border-inline-start: calc(var(--border-width) * 3) solid var(--second);
		color: var(--muted-foreground);
	}
	.related {
		margin-top: var(--space-8);
		padding-top: var(--space-4);
		border-top: var(--border-width) solid var(--border);
	}
	.related h2 {
		margin: 0 0 var(--space-2);
		font-size: var(--text-heading);
	}
	.related ul {
		margin: 0;
		padding: 0;
		list-style: none;
	}
	.related a {
		display: inline-flex;
		align-items: center;
		min-height: 2.75rem;
		color: var(--link);
	}
</style>
