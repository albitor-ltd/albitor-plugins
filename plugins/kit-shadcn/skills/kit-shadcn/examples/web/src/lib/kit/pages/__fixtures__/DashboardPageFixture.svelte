<!-- Test support: DashboardPage in the shell, for dashboardpage.test.ts alone. Delete it with
     DashboardPage.svelte and dashboardpage.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import Inbox from '@lucide/svelte/icons/inbox';
	import Card from '../../Card.svelte';
	import Table from '../../Table.svelte';
	import DashboardPage from '../DashboardPage.svelte';
	import PagesShowcase from './PagesShowcase.svelte';

	let {
		status = 'ready',
		firstRun = false
	}: { status?: 'loading' | 'error' | 'ready'; firstRun?: boolean } = $props();
	const noop = () => {};
</script>

<PagesShowcase>
	<DashboardPage
		title="Overview"
		stats={[
			{
				label: 'Open items',
				value: 12,
				icon: Inbox,
				delta: { direction: 'up', text: '3 more than last week', good: false }
			},
			{ label: 'Completed', value: 48, meter: { value: 48, label: 'Share completed' } }
		]}
		{firstRun}
		firstRunTitle="Nothing to show yet"
		firstRunDescription="Create your first report and its figures appear here."
		firstRunIcon={Inbox}
		{status}
		onretry={noop}
	>
		<Card title="Recent activity">
			<Table caption="Recent activity" columns={[{ key: 'what', label: 'What' }]} rows={[{ what: 'Report published' }]} />
		</Card>
	</DashboardPage>
</PagesShowcase>
