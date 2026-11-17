<!-- Test support: StatTile in the shell, for stattile.test.ts alone. Delete it with StatTile.svelte
     and stattile.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import Inbox from '@lucide/svelte/icons/inbox';
	import AppShell from '../AppShell.svelte';
	import StatTile from '../StatTile.svelte';
	import { fixtureNav } from './nav';
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1>Overview</h1>
	<StatTile
		label="Open items"
		value="12"
		icon={Inbox}
		delta={{ direction: 'up', text: '3 more than last week', good: false }}
	/>
	<StatTile label="Storage" value="62%" meter={{ value: 62, label: 'Storage used' }} />
	<StatTile
		label="Signups"
		value={41}
		spark={{
			label: 'Signups, last 3 days',
			data: [
				{ label: 'Mon', value: 4 },
				{ label: 'Tue', value: 7 },
				{ label: 'Wed', value: 30 }
			]
		}}
	/>
</AppShell>
