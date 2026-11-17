<!-- Test support: FilterChip in the shell, for filterchip.test.ts alone. Delete it with
     FilterChip.svelte and filterchip.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import FilterChip from '../FilterChip.svelte';
	import { fixtureNav } from './nav';

	let open = $state(true);
	let mine = $state(false);
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1>Reports</h1>
	<div role="group" aria-label="Filters">
		<FilterChip bind:pressed={open}>Open only</FilterChip>
		<FilterChip bind:pressed={mine}>Mine</FilterChip>
	</div>
</AppShell>
