<!-- Test support: Skeleton in the shell, for skeleton.test.ts alone. Delete it with Skeleton.svelte
     and skeleton.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import Skeleton from '../Skeleton.svelte';
	import { fixtureNav } from './nav';
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1>Reports</h1>
	<div aria-busy="true">
		<p>Loading reports</p>
		<Skeleton height="1.25rem" width="60%" />
		<Skeleton circle height="2.5rem" width="2.5rem" />
	</div>
</AppShell>
