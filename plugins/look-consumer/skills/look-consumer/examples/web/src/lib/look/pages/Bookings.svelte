<!-- A personal collection, grid first, in this look: the person's own bookings as soft cards, a
     one-line summary and a large Add action. The first-run state uses the look's round `EmptyArt`
     above one plain sentence. A starting point, then yours: replace the bookings. Built on the
     kit's CollectionPage. -->
<script lang="ts">
	import Button from '../../kit/Button.svelte';
	import CollectionPage from '../../kit/pages/CollectionPage.svelte';
	import EmptyArt from '../EmptyArt.svelte';

	let {
		status = 'ready',
		bookings = [
			{ id: 'b1', href: '/bookings/b1', title: 'Pottery for beginners', subtitle: 'Tuesdays, 7pm', badge: { tone: 'success' as const, label: 'Confirmed' } },
			{ id: 'b2', href: '/bookings/b2', title: 'Community choir', subtitle: 'Thursdays, 7:30pm', badge: { tone: 'warning' as const, label: 'On the waiting list' } }
		]
	}: {
		status?: 'loading' | 'error' | 'ready';
		bookings?: Array<{
			id: string;
			href: string;
			title: string;
			subtitle: string;
			badge: { tone: 'success' | 'warning'; label: string };
		}>;
	} = $props();
	const noop = () => {};
</script>

<CollectionPage
	title="My bookings"
	summary={`${bookings.length} classes, ${bookings.filter((b) => b.badge.tone === 'warning').length} waiting`}
	items={bookings}
	{status}
	onretry={noop}
	emptyTitle="No bookings yet"
	emptyDescription="Find a class you like and book a place. It will show up here."
>
	{#snippet actions()}<Button onclick={noop}>Find a class</Button>{/snippet}
	{#snippet emptyArt()}<EmptyArt />{/snippet}
	{#snippet emptyAction()}<Button onclick={noop}>Find a class</Button>{/snippet}
</CollectionPage>
