<!-- A list page, in this look: airy rows, one primary action in the header, a search field and
     filter chips above the table, and a first-run state with an icon and the next step (FirstRun.svelte adds the look's art). A starting
     point, then yours: replace the columns, the rows and the filters. Needs the kit's ListPage
     (copy it from the kit-shadcn skill). -->
<script lang="ts">
	import FolderKanban from '@lucide/svelte/icons/folder-kanban';
	import Plus from '@lucide/svelte/icons/plus';
	import Icon from '../../kit/Icon.svelte';
	import Button from '../../kit/Button.svelte';
	import Field from '../../kit/Field.svelte';
	import FilterChip from '../../kit/FilterChip.svelte';
	import ListPage from '../../kit/pages/ListPage.svelte';

	let {
		status = 'ready',
		rows = [
			{ name: 'api-gateway', owner: 'Platform', last: '12 minutes ago', state: 'Live' },
			{ name: 'billing-worker', owner: 'Payments', last: '1 hour ago', state: 'Failed' },
			{ name: 'web-app', owner: 'Growth', last: '3 hours ago', state: 'Rolling out' }
		]
	}: {
		status?: 'loading' | 'error' | 'ready';
		rows?: Array<Record<string, string | number>>;
	} = $props();

	let search = $state('');
	let failedOnly = $state(false);
	const shown = $derived(
		rows.filter(
			(r) =>
				String(r.name).toLowerCase().includes(search.toLowerCase()) &&
				(!failedOnly || r.state === 'Failed')
		)
	);
	const noop = () => {};
</script>

<ListPage
	title="Projects"
	description="Every project and where its last deployment stands."
	caption="Projects"
	columns={[
		{ key: 'name', label: 'Name' },
		{ key: 'state', label: 'Status' },
		{ key: 'owner', label: 'Team', priority: 'secondary' },
		{ key: 'last', label: 'Last deployed', priority: 'secondary' }
	]}
	rows={shown}
	narrow="priority"
	{status}
	onretry={noop}
	emptyTitle={rows.length === 0 ? 'No projects yet' : 'No projects match'}
	emptyDescription={rows.length === 0
		? 'Create a project to start deploying.'
		: 'Clear the search or the filter to see every project.'}
	emptyIcon={FolderKanban}
>
	{#snippet actions()}
		<Button onclick={noop}><Icon icon={Plus} size={16} /> New project</Button>
	{/snippet}
	{#snippet filters()}
		<Field label="Search projects" type="search" bind:value={search} />
		<FilterChip bind:pressed={failedOnly}>Failed</FilterChip>
	{/snippet}
	{#snippet emptyAction()}
		{#if rows.length === 0}<Button onclick={noop}>Create a project</Button>{/if}
	{/snippet}
</ListPage>
