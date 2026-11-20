<!-- Test support: CollectionPage in the shell, for collectionpage.test.ts alone. Delete it with
     CollectionPage.svelte and collectionpage.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import Sprout from '@lucide/svelte/icons/sprout';
	import Button from '../../Button.svelte';
	import CollectionPage from '../CollectionPage.svelte';
	import PagesShowcase from './PagesShowcase.svelte';

	let {
		status = 'ready',
		empty = false
	}: { status?: 'loading' | 'error' | 'ready'; empty?: boolean } = $props();

	const items = [
		{ id: 'fern', href: '/plants/fern', title: 'Boston fern', subtitle: 'Kitchen window', badge: { tone: 'warning', label: 'Needs water' } },
		{ id: 'basil', href: '/plants/basil', title: 'Basil', subtitle: 'Sill' }
	] as const;
	const noop = () => {};
</script>

<PagesShowcase>
	<CollectionPage
		title="My plants"
		summary="2 plants, 1 needs water"
		items={empty ? [] : [...items]}
		{status}
		onretry={noop}
		emptyTitle="No plants yet"
		emptyDescription="Add the first one to start your log."
		emptyIcon={Sprout}
	>
		{#snippet actions()}<Button onclick={noop}>Add a plant</Button>{/snippet}
		{#snippet emptyAction()}<Button onclick={noop}>Add a plant</Button>{/snippet}
	</CollectionPage>
</PagesShowcase>
