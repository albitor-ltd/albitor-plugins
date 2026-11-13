<!-- Test support: AppShell on its own, for the shell tests in kit.test.ts and rootclasses.test.ts and
     the shell checks. Every app has the shell, so this fixture and those tests stay.
     Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import { fixtureNav } from './nav';

	let { path = '/', extraNav }: { path?: string; extraNav?: string } = $props();
	const nav = $derived(fixtureNav(extraNav));
</script>

<AppShell {nav} {path} title="Example app">
	<h1 class="govuk-heading-l">Reports</h1>
	<p class="govuk-body">The page content.</p>
</AppShell>
