<!-- Test support: EmptyState in the shell, for emptystate.test.ts alone. Delete it with
     EmptyState.svelte and emptystate.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import EmptyState from '../EmptyState.svelte';
	import { fixtureNav } from './nav';
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1>Reports</h1>
	<EmptyState title="No reports yet" description="Reports you create appear here." />
</AppShell>
