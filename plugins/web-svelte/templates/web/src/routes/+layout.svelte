<!-- Managed by Albitor (web-svelte): the slot host. Providers wrap the kit's app shell, which
     receives the nav. Per-area chrome belongs in a route group's own +layout.svelte. -->
<script lang="ts">
	import '../lib/app/app.css';
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';
	import AppShell from '../lib/kit/AppShell.svelte';
	import OfflineBanner from '../lib/shell/OfflineBanner.svelte';
	import ShellProviders from '../lib/shell/ShellProviders.svelte';
	import { navGroups } from '../lib/shell/nav.generated';
	import { providers } from '../lib/shell/providers.generated';

	let { children }: { children: Snippet } = $props();
</script>

<ShellProviders {providers}>
	<AppShell nav={navGroups} path={page.url.pathname}>
		<OfflineBanner />
		{@render children()}
	</AppShell>
</ShellProviders>
