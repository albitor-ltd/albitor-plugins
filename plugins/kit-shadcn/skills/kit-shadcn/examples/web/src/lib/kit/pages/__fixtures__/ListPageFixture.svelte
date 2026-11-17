<!-- Test support: ListPage in the shell, for listpage.test.ts alone. Delete it with ListPage.svelte
     and listpage.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import Inbox from '@lucide/svelte/icons/inbox';
	import Button from '../../Button.svelte';
	import Field from '../../Field.svelte';
	import FilterChip from '../../FilterChip.svelte';
	import type { Narrow } from '../../Table.svelte';
	import ListPage from '../ListPage.svelte';
	import PagesShowcase from './PagesShowcase.svelte';

	let {
		status = 'ready',
		rows = [{ name: 'Quarterly summary', owner: 'Finance' }],
		narrow = 'scroll'
	}: {
		status?: 'loading' | 'error' | 'ready';
		rows?: Array<Record<string, string | number>>;
		narrow?: Narrow;
	} = $props();

	let search = $state('');
	let onlyMine = $state(false);
	const noop = () => {};
</script>

<PagesShowcase>
	<ListPage
		title="Reports"
		description="Everything your team has written."
		caption="Reports"
		columns={[
			{ key: 'name', label: 'Name' },
			{ key: 'owner', label: 'Owner', priority: 'secondary' }
		]}
		{rows}
		{narrow}
		{status}
		onretry={noop}
		emptyDescription="Create the first report to see it here."
		emptyIcon={Inbox}
	>
		{#snippet actions()}<Button onclick={noop}>New report</Button>{/snippet}
		{#snippet filters()}
			<Field label="Search reports" type="search" bind:value={search} />
			<FilterChip bind:pressed={onlyMine}>Mine</FilterChip>
		{/snippet}
		{#snippet emptyAction()}<Button onclick={noop}>New report</Button>{/snippet}
	</ListPage>
</PagesShowcase>
