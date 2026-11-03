<!-- The slot host. Providers wrap the kit's app shell, which
     receives the nav and the app's name (the platform's app_name token, sanitised for a quoted
     attribute). `data.showNav` (from `+layout.ts`) hides the nav, and the AccountBar, from a
     signed-out visitor. Per-area chrome belongs in a route group's own +layout.svelte. -->
<script lang="ts">
	import '../lib/app/app.css';
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';
	import type { LayoutData } from './$types';
	import AccountBar from '../lib/shell/AccountBar.svelte';
	import AppShell from '../lib/kit/AppShell.svelte';
	import OfflineBanner from '../lib/shell/OfflineBanner.svelte';
	import ShellProviders from '../lib/shell/ShellProviders.svelte';
	import { navGroups } from '../lib/shell/nav.generated';
	import { providers } from '../lib/shell/providers.generated';

	let { children, data }: { children: Snippet; data: LayoutData } = $props();
</script>

<ShellProviders {providers}>
	<AppShell nav={data.showNav ? navGroups : []} path={page.url.pathname} title="{{albitor.app_name}}">
		<OfflineBanner />
		{#if data.showNav}
			<AccountBar />
		{/if}
		{@render children()}
	</AppShell>
</ShellProviders>
