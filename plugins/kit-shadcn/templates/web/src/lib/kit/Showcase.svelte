<!-- Seeded by Albitor (kit-shadcn), then yours: every shell piece on one page, and what kit.test.ts
     audits with axe. Copy from it when building a page; delete it when you no longer need it
     (and kit.test.ts with it). Not routed: nothing links here. -->
<script lang="ts">
	import AppShell from './AppShell.svelte';
	import Button from './Button.svelte';
	import Dialog from './Dialog.svelte';
	import EmptyState from './EmptyState.svelte';
	import Field from './Field.svelte';
	import Table from './Table.svelte';
	import type { NavGroup } from '../shell/types';

	let {
		path = '/',
		dialogOpen = false,
		extraNav,
		rowName = 'Quarterly summary',
		empty = false
	}: {
		path?: string;
		dialogOpen?: boolean;
		extraNav?: string;
		rowName?: string;
		empty?: boolean;
	} = $props();

	const nav: NavGroup[] = $derived([
		{
			id: null,
			items: [
				{ label: 'Home', order: 0, href: '/' },
				{ label: 'Reports', order: 10, href: '/reports' },
				...(extraNav ? [{ label: extraNav, order: 20, href: '/extra' }] : [])
			]
		}
	]);

	// svelte-ignore state_referenced_locally
	let open = $state(dialogOpen);
	let email = $state('');
</script>

<AppShell {nav} {path} title="Example app">
	<h1>Reports</h1>

	<form onsubmit={(e) => e.preventDefault()}>
		<Field
			label="Email address"
			type="email"
			bind:value={email}
			autocomplete="email"
			hint="We only use it to send the report."
			error="Enter an email address, like name@example.com"
		/>
		<Field label="Notes" multiline />
		<Button type="submit">Send report</Button>
		<Button variant="destructive" onclick={() => (open = true)}>Delete report</Button>
	</form>

	{#if empty}
		<EmptyState title="No reports yet" description="Reports you create appear here." />
	{:else}
		<Table
			caption="Recent reports"
			columns={[
				{ key: 'name', label: 'Name' },
				{ key: 'owner', label: 'Owner' }
			]}
			rows={[{ name: rowName, owner: 'Finance' }]}
		/>
	{/if}

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
