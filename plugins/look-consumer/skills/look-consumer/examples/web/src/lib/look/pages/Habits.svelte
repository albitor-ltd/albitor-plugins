<!-- A list page, in this look: one column of big rows, each record stacked into a labelled card on
     a phone, filter chips, and the one main action in the header. A starting point, then yours:
     replace the columns, the rows and the filters. Needs the kit's ListPage (copy it from the
     kit-shadcn skill). -->
<script lang="ts">
	import ListChecks from '@lucide/svelte/icons/list-checks';
	import Plus from '@lucide/svelte/icons/plus';
	import Button from '../../kit/Button.svelte';
	import FilterChip from '../../kit/FilterChip.svelte';
	import Icon from '../../kit/Icon.svelte';
	import ListPage from '../../kit/pages/ListPage.svelte';

	let {
		status = 'ready',
		rows = [
			{ name: 'Evening walk', how: 'Every day', streak: '12 days' },
			{ name: 'Read for 20 minutes', how: 'Weekdays', streak: '5 days' },
			{ name: 'Water the plants', how: 'Twice a week', streak: '3 weeks' }
		]
	}: {
		status?: 'loading' | 'error' | 'ready';
		rows?: Array<Record<string, string | number>>;
	} = $props();

	let daily = $state(false);
	const shown = $derived(daily ? rows.filter((r) => r.how === 'Every day') : rows);
	const noop = () => {};
</script>

<ListPage
	title="Habits"
	description="Everything you are keeping up."
	caption="Habits"
	columns={[
		{ key: 'name', label: 'Habit' },
		{ key: 'how', label: 'How often' },
		{ key: 'streak', label: 'Streak' }
	]}
	rows={shown}
	narrow="stacked"
	{status}
	onretry={noop}
	emptyTitle={rows.length === 0 ? 'No habits yet' : 'No habits match'}
	emptyDescription={rows.length === 0
		? 'Add a habit to start a streak.'
		: 'Clear the filter to see every habit.'}
	emptyIcon={ListChecks}
>
	{#snippet actions()}
		<Button onclick={noop}><Icon icon={Plus} size={20} /> Add habit</Button>
	{/snippet}
	{#snippet filters()}
		<FilterChip bind:pressed={daily}>Every day</FilterChip>
	{/snippet}
	{#snippet emptyAction()}
		{#if rows.length === 0}<Button onclick={noop}>Add a habit</Button>{/if}
	{/snippet}
</ListPage>
