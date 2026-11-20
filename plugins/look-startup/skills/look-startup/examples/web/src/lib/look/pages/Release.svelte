<!-- An article, in this look: a release note. A link back to all releases, the title, a one-line
     standfirst, the date as the byline, then the body at reading width, with `code` and branch
     names in the monospace face. Generous whitespace, headings from the heading size. A starting
     point, then yours: replace the copy. Built on the kit's ArticlePage. -->
<script lang="ts">
	import ArticlePage from '../../kit/pages/ArticlePage.svelte';

	let { status = 'ready' }: { status?: 'loading' | 'error' | 'ready' } = $props();
	const noop = () => {};
</script>

<ArticlePage
	title="November release: faster previews"
	description="Branch previews now start in under ten seconds."
	back={{ href: '/releases', label: 'All releases' }}
	byline="The product team"
	published="2026-11-20"
	readingTime="2 min read"
	related={[
		{ href: '/releases/2026-10', label: 'October release: rollbacks' },
		{ href: '/docs/previews', label: 'Previewing a branch' }
	]}
	{status}
	onretry={noop}
>
	<p>A preview used to wait for a full build. It now reuses the build cache from <code>main</code>.</p>
	<h2>What changed</h2>
	<ul>
		<li>Previews start in under ten seconds for most projects.</li>
		<li>The preview address is shown on the pull request.</li>
	</ul>
	<h2>What you need to do</h2>
	<p>Nothing. Existing projects switch over on their next push.</p>
</ArticlePage>

<style>
	:global(.prose code) {
		font-family: var(--font-mono, ui-monospace, monospace);
		font-size: 0.9em;
	}
</style>
