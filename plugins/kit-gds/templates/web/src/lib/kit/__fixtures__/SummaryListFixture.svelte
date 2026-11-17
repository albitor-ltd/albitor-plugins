<!-- Test support: SummaryList in the shell, for summarylist.test.ts alone. Delete it with SummaryList.svelte and summarylist.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import SummaryList from '../SummaryList.svelte';
	import SummaryCard from '../SummaryCard.svelte';
	import { fixtureNav } from './nav';

	let { withChange = true }: { withChange?: boolean } = $props();

	const rows = $derived([
		{ key: 'Name', value: 'Sarah Philips', change: withChange ? { href: '/name', hidden: 'name' } : undefined },
		{ key: 'Email', value: 'sarah@example.com', change: withChange ? { href: '/email', hidden: 'email' } : undefined }
	]);
</script>

<AppShell nav={fixtureNav()} path="/reports" title="Example app">
	<h1 class="govuk-heading-l">Check your answers</h1>
	<SummaryList {rows} />
	<SummaryCard
		title="Second applicant"
		actions={[{ href: '/applicants/2/delete', text: 'Delete' }, { href: '/applicants/2', text: 'Change' }]}
		{rows}
	/>
	<SummaryCard title="Third applicant" actions={[{ href: '/applicants/3/delete', text: 'Delete' }]} {rows} />
</AppShell>
