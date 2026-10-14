<!-- Managed by Albitor (web-svelte): nests the web.providers components, outermost first. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Self from './ShellProviders.svelte';
	import type { ProviderComponent } from './types';

	let { providers, children }: { providers: ProviderComponent[]; children: Snippet } = $props();

	const Head = $derived(providers[0]);
	const rest = $derived(providers.slice(1));
</script>

{#if Head}
	<Head>
		<Self providers={rest} {children} />
	</Head>
{:else}
	{@render children()}
{/if}
