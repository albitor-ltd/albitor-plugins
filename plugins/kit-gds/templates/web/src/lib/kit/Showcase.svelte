<!-- Starting point (kit-gds): every shell piece on one page, and what kit.test.ts and
     dialog.test.ts audit with axe. Copy from it when building a page. Keep it: it is what the audits
     run against; it is not routed and nothing links here. -->
<script lang="ts">
	import AppShell from './AppShell.svelte';
	import Button from './Button.svelte';
	import Dialog from './Dialog.svelte';
	import ErrorSummary from './ErrorSummary.svelte';
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
	<ErrorSummary errors={[{ field: 'email', message: 'Enter an email address, like name@example.com' }]} />

	<h1 class="govuk-heading-l">Reports</h1>

	<form onsubmit={(e) => e.preventDefault()}>
		<Field
			id="email"
			label="Email address"
			type="email"
			bind:value={email}
			autocomplete="email"
			hint="We only use it to send the report."
			error="Enter an email address, like name@example.com"
		/>
		<Field label="Notes" multiline aria-describedby="notes-help" />
		<p class="govuk-body" id="notes-help">Notes are visible to your team.</p>
		<div class="govuk-button-group">
			<Button type="submit">Send report</Button>
			<Button variant="destructive" onclick={() => (open = true)}>Delete report</Button>
		</div>
	</form>

	<Table
		caption="Recent reports"
		columns={[
			{ key: 'name', label: 'Name' },
			{ key: 'owner', label: 'Owner' }
		]}
		rows={empty ? [] : [{ name: rowName, owner: 'Finance' }]}
		emptyTitle="No reports yet"
		emptyDescription="Reports you create appear here."
	/>

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
