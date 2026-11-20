<!-- Test support: ArticlePage in the shell, for articlepage.test.ts alone. Delete it with
     ArticlePage.svelte and articlepage.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import ArticlePage from '../ArticlePage.svelte';
	import PagesShowcase from './PagesShowcase.svelte';

	let { status = 'ready' }: { status?: 'loading' | 'error' | 'ready' } = $props();
	const noop = () => {};
</script>

<PagesShowcase>
	<ArticlePage
		title="Winter opening hours"
		description="What changes from 1 December."
		back={{ href: '/news', label: 'All news' }}
		byline="Sam Rowe"
		published="2026-11-20"
		readingTime="2 min read"
		related={[{ href: '/news/parking', label: 'Parking at the hall' }]}
		{status}
		onretry={noop}
	>
		<p>From 1 December the hall opens at 9am.</p>
		<h2>Weekends</h2>
		<p>Weekend hours stay the same.</p>
		<blockquote>We would rather open late than close early.</blockquote>
	</ArticlePage>
</PagesShowcase>
