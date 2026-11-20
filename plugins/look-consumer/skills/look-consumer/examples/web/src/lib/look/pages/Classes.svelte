<!-- A catalogue, in this look: search and filter neighbourhood classes, each a soft card with its
     time and a status, then open one. Large targets, friendly copy, the search and the chips on
     one row that wraps on a phone. No imagery: the cards are type and a Pill. A starting point,
     then yours: replace the classes and the filters. Built on the kit's CataloguePage. -->
<script lang="ts">
	import Palette from '@lucide/svelte/icons/palette';
	import Field from '../../kit/Field.svelte';
	import FilterChip from '../../kit/FilterChip.svelte';
	import CataloguePage from '../../kit/pages/CataloguePage.svelte';

	type Item = {
		id: string;
		href: string;
		title: string;
		summary: string;
		meta: string;
		badge?: { tone: 'success' | 'warning'; label: string };
		evening: boolean;
	};

	let {
		status = 'ready',
		classes = [
			{
				id: 'pottery',
				href: '/classes/pottery',
				title: 'Pottery for beginners',
				summary: 'Six relaxed weeks at the wheel. Everything provided.',
				meta: 'Tuesdays, 7pm',
				badge: { tone: 'success', label: 'Places left' },
				evening: true
			},
			{
				id: 'sketching',
				href: '/classes/sketching',
				title: 'Urban sketching',
				summary: 'Draw the town with a small group and a flask of tea.',
				meta: 'Saturdays, 10am',
				badge: { tone: 'warning', label: 'Nearly full' },
				evening: false
			},
			{
				id: 'choir',
				href: '/classes/choir',
				title: 'Community choir',
				summary: 'No audition. Bring a friend.',
				meta: 'Thursdays, 7:30pm',
				evening: true
			}
		] as Item[]
	}: { status?: 'loading' | 'error' | 'ready'; classes?: Item[] } = $props();

	let query = $state('');
	let evenings = $state(false);
	const matches = $derived(
		classes.filter(
			(c) =>
				(c.title + c.summary).toLowerCase().includes(query.toLowerCase()) && (!evenings || c.evening)
		)
	);
	function clear() {
		query = '';
		evenings = false;
	}
	const noop = () => {};
</script>

<CataloguePage
	title="Classes near you"
	description="Short courses in the town hall and the library."
	items={matches}
	total={classes.length}
	{status}
	onretry={noop}
	onclear={clear}
	emptyTitle="No classes yet"
	emptyDescription="New classes are added each term."
	emptyIcon={Palette}
>
	{#snippet filters()}
		<Field label="Search classes" type="search" bind:value={query} />
		<FilterChip bind:pressed={evenings}>Evenings</FilterChip>
	{/snippet}
</CataloguePage>
