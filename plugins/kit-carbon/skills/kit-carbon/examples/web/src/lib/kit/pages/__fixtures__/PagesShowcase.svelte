<!-- Test support: the frame the page fixtures render inside, AppShell with the example nav. It is
     not a showcase of the pages: each page has its own fixture beside this one and its own test,
     so an app that never uses a page template deletes the template, its fixture and its test
     together. The name stays because the kit certification looks for PagesShowcase.svelte in a
     kit's page examples. Not routed; nothing links here. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import AppShell from '../../AppShell.svelte';
	import { fixtureNav } from '../../__fixtures__/nav';

	let { children }: { children: Snippet } = $props();
</script>

<AppShell nav={fixtureNav()} path="/reports" title="Example app">
	{@render children()}
</AppShell>
