<!-- Managed by Albitor (client-online): the web.providers entry that gives every page the shared
     query client. One client per page load, created here and not at module level. -->
<script lang="ts">
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import type { Snippet } from 'svelte';
	import { createQueryClient } from './client';

	let { children }: { children: Snippet } = $props();

	const client = createQueryClient();
</script>

<QueryClientProvider {client}>
	{@render children()}
</QueryClientProvider>
