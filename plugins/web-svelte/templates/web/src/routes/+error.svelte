<!-- Seeded by Albitor (web-svelte), then yours: the error and 404 page. -->
<script lang="ts">
	import { page } from '$app/state';
</script>

<svelte:head>
	<title>{page.status === 404 ? 'Page not found' : 'Something went wrong'}</title>
</svelte:head>

<section>
	{#if page.status === 404}
		<h1>Page not found</h1>
		<p>There is nothing at this address. It may have moved, or the link may be wrong.</p>
	{:else}
		<h1>Something went wrong</h1>
		<p>{page.error?.message ?? 'An unexpected error stopped this page from loading.'}</p>
	{/if}
	<p><a href="/">Back to the start</a></p>
</section>
