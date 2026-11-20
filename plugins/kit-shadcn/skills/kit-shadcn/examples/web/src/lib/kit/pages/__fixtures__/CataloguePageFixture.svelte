<!-- Test support: CataloguePage in the shell, for cataloguepage.test.ts alone. Delete it with
     CataloguePage.svelte and cataloguepage.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import BookOpen from '@lucide/svelte/icons/book-open';
	import Field from '../../Field.svelte';
	import FilterChip from '../../FilterChip.svelte';
	import CataloguePage from '../CataloguePage.svelte';
	import PagesShowcase from './PagesShowcase.svelte';

	const all = [
		{
			id: 'pottery',
			href: '/courses/pottery',
			title: 'Pottery for beginners',
			summary: 'Six weeks at the wheel.',
			meta: 'Tuesdays, 7pm',
			badge: { tone: 'success', label: 'Places left' }
		},
		{ id: 'sketching', href: '/courses/sketching', title: 'Urban sketching', summary: 'Draw the town.' }
	] as const;

	let {
		status = 'ready',
		matches = 'all',
		total = 2
	}: { status?: 'loading' | 'error' | 'ready'; matches?: 'all' | 'one' | 'none'; total?: number } = $props();

	let query = $state('');
	let evenings = $state(false);
	const items = $derived(matches === 'all' ? [...all] : matches === 'one' ? [all[0]] : []);
	const noop = () => {};
</script>

<PagesShowcase>
	<CataloguePage
		title="Courses"
		description="Evening and weekend classes."
		{items}
		{total}
		{status}
		onretry={noop}
		onclear={noop}
		emptyTitle="No courses yet"
		emptyDescription="Courses are added each term."
		emptyIcon={BookOpen}
	>
		{#snippet filters()}
			<Field label="Search courses" type="search" bind:value={query} />
			<FilterChip bind:pressed={evenings}>Evenings</FilterChip>
		{/snippet}
	</CataloguePage>
</PagesShowcase>
