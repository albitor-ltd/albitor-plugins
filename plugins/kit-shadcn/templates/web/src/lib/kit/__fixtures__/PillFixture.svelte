<!-- Test support: Pill in the shell, for pill.test.ts alone. Delete it with Pill.svelte and
     pill.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import Pill from '../Pill.svelte';
	import { fixtureNav } from './nav';
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1>Status</h1>
	<Pill>Draft</Pill>
	<Pill tone="success">Live</Pill>
	<Pill tone="warning">Needs review</Pill>
	<Pill tone="danger">Failed</Pill>
	<Pill tone="info">New</Pill>
</AppShell>
