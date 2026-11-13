<!-- Test support: Dialog in the shell, for dialog.test.ts alone. Delete it with Dialog.svelte and
     dialog.test.ts. Not routed; nothing links here. -->
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
	<h1 class="govuk-heading-l">Reports</h1>
	<div class="govuk-button-group">
		<Button variant="destructive" onclick={() => (open = true)}>Delete report</Button>
	</div>

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
