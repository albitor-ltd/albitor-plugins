<!-- Test support: HomePage in the shell, for homepage.test.ts alone. Delete it with HomePage.svelte
     and homepage.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import Clock from '@lucide/svelte/icons/clock';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import HomePage from '../HomePage.svelte';
	import PagesShowcase from './PagesShowcase.svelte';

	let { bare = false }: { bare?: boolean } = $props();
</script>

<PagesShowcase>
	{#if bare}
		<HomePage title="Book a pitch" lead="See what is free." primary={{ href: '/pitches', label: 'See free pitches' }} />
	{:else}
		<HomePage
			title="Book a pitch at the club"
			lead="See what is free and reserve it in a minute."
			primary={{ href: '/pitches', label: 'See free pitches' }}
			secondary={{ href: '/how', label: 'How it works' }}
			highlights={[
				{ title: 'Live availability', text: 'Slots update as they are booked.', icon: Clock },
				{ title: 'Close to home', text: 'Four grounds across the town.', icon: MapPin }
			]}
			closing={{
				title: 'Ready to play?',
				text: 'Pick a ground and a time.',
				action: { href: '/pitches', label: 'Book now' }
			}}
		>
			<section>
				<h2>How it works</h2>
				<p>Choose a ground, choose a time, confirm.</p>
			</section>
		</HomePage>
	{/if}
</PagesShowcase>
