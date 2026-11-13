<!-- Test support: AppShell on its own, for the shell tests in kit.test.ts. Every app has the shell
     and the toaster it renders, so this fixture and those tests stay. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import { fixtureNav } from './nav';

	let { path = '/', extraNav }: { path?: string; extraNav?: string } = $props();
	const nav = $derived(fixtureNav(extraNav));
</script>

<AppShell {nav} {path} title="Example app">
	<h1>Reports</h1>
	<p>The page content.</p>
</AppShell>
