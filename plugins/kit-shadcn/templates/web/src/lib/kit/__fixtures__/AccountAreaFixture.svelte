<!-- Test support: AccountArea in the shell's account slot, for accountarea.test.ts alone. Delete it
     with AccountArea.svelte and accountarea.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AccountArea, { type SaveStatus } from '../AccountArea.svelte';
	import AppShell from '../AppShell.svelte';
	import { fixtureNav } from './nav';

	let {
		saveStatus = 'saved',
		theme = 'light'
	}: {
		saveStatus?: SaveStatus;
		theme?: 'light' | 'dark';
	} = $props();
</script>

{#snippet account()}
	<AccountArea
		name="Ada Lovelace"
		{theme}
		onthemechange={() => {}}
		{saveStatus}
		onsignout={() => {}}
	/>
{/snippet}

<AppShell nav={fixtureNav()} path="/" title="Example app" {account}>
	<h1>Home</h1>
</AppShell>
