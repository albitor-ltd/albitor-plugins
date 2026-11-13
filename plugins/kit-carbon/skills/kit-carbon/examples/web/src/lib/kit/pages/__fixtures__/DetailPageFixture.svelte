<!-- Test support: DetailPage in the shell, for detailpage.test.ts alone. Delete it with DetailPage.svelte
     and detailpage.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import Button from '../../Button.svelte';
	import Table from '../../Table.svelte';
	import DetailPage from '../DetailPage.svelte';
	import PagesShowcase from './PagesShowcase.svelte';

	let {
		status = 'ready'
	}: {
		status?: 'loading' | 'error' | 'ready';
	} = $props();

	const noop = () => {};
</script>

<PagesShowcase>
	<DetailPage
		title="Quarterly summary"
		back={{ href: '/reports', label: 'All reports' }}
		fields={[
			{ label: 'Owner', value: 'Finance' },
			{ label: 'Status', value: 'Published' },
			{ label: 'Notes', value: '' }
		]}
		{status}
		onretry={noop}
	>
		{#snippet actions()}<Button variant="destructive" onclick={noop}>Delete</Button>{/snippet}
		<h2>Revisions</h2>
		<Table caption="Revisions" columns={[{ key: 'when', label: 'When' }]} rows={[{ when: 'Today' }]} />
	</DetailPage>
</PagesShowcase>
