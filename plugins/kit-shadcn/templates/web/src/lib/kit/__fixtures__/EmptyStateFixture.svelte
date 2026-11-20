<!-- Test support: EmptyState in the shell, for emptystate.test.ts alone. Delete it with
     EmptyState.svelte and emptystate.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import EmptyState from '../EmptyState.svelte';
	import { fixtureNav } from './nav';

	let { art: withArt = false }: { art?: boolean } = $props();
</script>

{#snippet drawing()}
	<svg width="96" height="96" viewBox="0 0 160 160" aria-hidden="true" focusable="false">
		<rect x="36" y="24" width="88" height="112" />
	</svg>
{/snippet}

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1>Reports</h1>
	<EmptyState
		title="No reports yet"
		description="Reports you create appear here."
		art={withArt ? drawing : undefined}
	/>
</AppShell>
