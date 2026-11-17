<!-- Test support: AppShell with a chosen number of destinations, phone stance and layout, for
     navshell.test.ts alone. Delete it with NavShell.svelte and navshell.test.ts. Not routed;
     nothing links here. -->
<script lang="ts">
	import House from '@lucide/svelte/icons/house';
	import AppShell from '../AppShell.svelte';
	import type { NavGroup } from '../../shell/types';

	let {
		count = 3,
		path = '/',
		phoneNav = 'drawer',
		layout = 'top',
		withIcons = false
	}: {
		count?: number;
		path?: string;
		phoneNav?: 'drawer' | 'tabs';
		layout?: 'top' | 'side';
		withIcons?: boolean;
	} = $props();

	const nav: NavGroup[] = $derived([
		{
			id: null,
			items: Array.from({ length: count }, (_, i) => ({
				label: `Page ${i + 1}`,
				order: i,
				href: i === 0 ? '/' : `/p${i + 1}`
			}))
		}
	]);
	const icons = $derived(withIcons ? { '/': House } : undefined);
</script>

<AppShell {nav} {path} {phoneNav} {layout} {icons} title="Example app">
	<h1>Page</h1>
	<p>The page content.</p>
</AppShell>
