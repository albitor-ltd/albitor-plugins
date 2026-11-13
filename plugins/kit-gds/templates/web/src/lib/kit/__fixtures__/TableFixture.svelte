<!-- Test support: Table in the shell, for table.test.ts alone. Delete it with Table.svelte and
     table.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import Table from '../Table.svelte';
	import { fixtureNav } from './nav';

	let { rowName = 'Quarterly summary', empty = false }: { rowName?: string; empty?: boolean } = $props();
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1 class="govuk-heading-l">Reports</h1>
	<Table
		caption="Recent reports"
		columns={[
			{ key: 'name', label: 'Name' },
			{ key: 'owner', label: 'Owner' }
		]}
		rows={empty ? [] : [{ name: rowName, owner: 'Finance' }]}
		emptyTitle="No reports yet"
		emptyDescription="Reports you create appear here."
	/>
</AppShell>
