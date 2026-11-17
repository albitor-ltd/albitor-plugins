<!-- Test support: Card in the shell, for card.test.ts alone. Delete it with Card.svelte and
     card.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import Card from '../Card.svelte';
	import { fixtureNav } from './nav';
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1>Overview</h1>
	<Card title="Recent activity">
		{#snippet actions()}<a href="/reports">View all</a>{/snippet}
		<p>Three reports were published.</p>
	</Card>
	<Card tone="second"><p>A note without a heading.</p></Card>
</AppShell>
