<!-- Test support: Dialog in the shell, for dialog.test.ts alone. Delete it with Dialog.svelte and
     dialog.test.ts. The email input is only something behind the dialog for the focus-trap test.
     Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import Button from '../Button.svelte';
	import Dialog from '../Dialog.svelte';
	import { fixtureNav } from './nav';

	let { dialogOpen = false }: { dialogOpen?: boolean } = $props();

	// svelte-ignore state_referenced_locally
	let open = $state(dialogOpen);
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1>Reports</h1>
	<form onsubmit={(e) => e.preventDefault()}>
		<label>Email address <input type="email" autocomplete="email" /></label>
		<Button variant="destructive" onclick={() => (open = true)}>Delete report</Button>
	</form>

	<Dialog
		bind:open
		title="Delete report"
		description="This deletes the report for everyone and cannot be undone."
	>
		{#snippet actions()}
			<Button variant="destructive" onclick={() => (open = false)}>Delete</Button>
		{/snippet}
	</Dialog>
</AppShell>
