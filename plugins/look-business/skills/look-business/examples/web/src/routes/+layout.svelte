<!-- The `layout` and `phoneNav` values are this look's fallback: the app's structure pick sets them
     when it names a shell (see the kit-shadcn SKILL).
     The root layout with this look's shell wiring: a compact sidebar, the Menu button and drawer on
     a phone (this look keeps the dense, table-first screen uncluttered) and an icon for each
     destination. It is web-svelte's layout plus three props on AppShell (`layout`, `phoneNav` and
     `icons`): keep the layout the app already has (its providers, its title and mark) and add only
     those. Structure (layout, phoneNav) and style (the look's tokens) are separate: change one
     without the other.

     `icons` is keyed by each destination's href, so list the app's own routes here and update the
     map whenever a route is added, renamed or removed. A destination with no entry shows its label
     alone. Import each icon on its own from `@lucide/svelte/icons/<name>`, never the barrel. -->
<script lang="ts">
	import '../lib/app/app.css';
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';
	import type { LayoutData } from './$types';
	import LayoutDashboard from '@lucide/svelte/icons/layout-dashboard';
	import Receipt from '@lucide/svelte/icons/receipt';
	import Settings from '@lucide/svelte/icons/settings';
	import Users from '@lucide/svelte/icons/users';
	import AccountBar from '../lib/shell/AccountBar.svelte';
	import AppShell from '../lib/kit/AppShell.svelte';
	import OfflineBanner from '../lib/shell/OfflineBanner.svelte';
	import ShellProviders from '../lib/shell/ShellProviders.svelte';
	import { navGroups } from '../lib/shell/nav.generated';
	import { providers } from '../lib/shell/providers.generated';

	let { children, data }: { children: Snippet; data: LayoutData } = $props();

	// href to icon, one entry per destination in the app's nav.
	const icons = {
		'/': LayoutDashboard,
		'/invoices': Receipt,
		'/suppliers': Users,
		'/settings': Settings
	};
</script>

<ShellProviders {providers}>
	<AppShell
		nav={data.showNav ? navGroups : []}
		path={page.url.pathname}
		title="Example app"
		mark="/brand/mark.svg"
		layout="side"
		phoneNav="drawer"
		{icons}
	>
		<OfflineBanner />
		{#if data.showNav}
			<AccountBar />
		{/if}
		{@render children()}
	</AppShell>
</ShellProviders>
