<!-- Starting point (web-svelte, online data layer): loading, offline, error, empty and ready for one query.
     Pass the query result; every state has a default, and each can be replaced by a snippet. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { errorMessage, viewState, type QueryLike } from './state';

	type Result = QueryLike & { error?: unknown; refetch?: () => unknown };

	let {
		query,
		isEmpty,
		loading,
		offline,
		error,
		empty,
		children,
		emptyText = 'Nothing here yet.'
	}: {
		query: Result;
		/** Decides when a successful result counts as empty. Defaults to no items. */
		isEmpty?: (data: unknown) => boolean;
		loading?: Snippet;
		offline?: Snippet;
		error?: Snippet<[message: string, retry: () => void]>;
		empty?: Snippet;
		/** Rendered with the data when the query is ready. */
		children: Snippet<[data: unknown]>;
		emptyText?: string;
	} = $props();

	const view = $derived(viewState(query, isEmpty));
	const message = $derived(errorMessage(query.error));
	const retry = () => void query.refetch?.();
</script>

{#if view === 'loading'}
	{#if loading}{@render loading()}{:else}<p role="status">Loading...</p>{/if}
{:else if view === 'offline'}
	{#if offline}{@render offline()}{:else}<p role="status">You are offline. This will load when you are back online.</p>{/if}
{:else if view === 'error'}
	{#if error}
		{@render error(message, retry)}
	{:else}
		<div role="alert">
			<p>{message}</p>
			{#if query.refetch}<button type="button" onclick={retry}>Try again</button>{/if}
		</div>
	{/if}
{:else if view === 'empty'}
	{#if empty}{@render empty()}{:else}<p>{emptyText}</p>{/if}
{:else}
	{@render children(query.data)}
{/if}
