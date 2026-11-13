<!-- Test support: DashboardPage in the shell, for dashboardpage.test.ts alone. Delete it with DashboardPage.svelte
     and dashboardpage.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import Table from '../../Table.svelte';
	import DashboardPage from '../DashboardPage.svelte';
	import PagesShowcase from './PagesShowcase.svelte';

	let {
		status = 'ready'
	}: {
		status?: 'loading' | 'error' | 'ready';
	} = $props();

	const noop = () => {};
</script>

<PagesShowcase>
	<DashboardPage
		title="Overview"
		stats={[
			{ label: 'Open items', value: 12, hint: '3 more than last week' },
			{ label: 'Completed', value: 48 }
		]}
		{status}
		onretry={noop}
	>
		<h2 class="govuk-heading-m">Recent activity</h2>
		<Table caption="Recent activity" columns={[{ key: 'what', label: 'What' }]} rows={[{ what: 'Report published' }]} />
	</DashboardPage>
</PagesShowcase>
