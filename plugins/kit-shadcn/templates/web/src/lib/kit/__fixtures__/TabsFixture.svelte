<!-- Test support: Tabs in the shell, for tabs.test.ts alone. Delete it with Tabs.svelte and
     tabs.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import Tabs from '../Tabs.svelte';
	import { fixtureNav } from './nav';
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1>Report</h1>
	<Tabs
		label="Report sections"
		tabs={[
			{ value: 'summary', label: 'Summary' },
			{ value: 'detail', label: 'Detail' },
			{ value: 'history', label: 'History' }
		]}
	>
		{#snippet panel(value)}<p>The {value} view.</p>{/snippet}
	</Tabs>
</AppShell>
