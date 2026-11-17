<!-- Test support: Chart in the shell, for chart.test.ts alone. Delete it with Chart.svelte and
     chart.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import Chart from '../Chart.svelte';
	import { fixtureNav } from './nav';

	const days = [
		{ label: 'Mon', value: 4 },
		{ label: 'Tue', value: 7 },
		{ label: 'Wed', value: 3 }
	];
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1>Orders</h1>
	<Chart title="Orders per day" data={days} unit=" orders" />
	<Chart kind="sparkline" compact title="Signups, last 3 days" data={days} />
	<Chart title="Nothing yet" data={[]} />
</AppShell>
