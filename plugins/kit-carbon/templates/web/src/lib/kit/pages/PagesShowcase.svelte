<!-- Managed by Albitor (kit-carbon): each page template in each state, rendered inside AppShell, and
     what pages.test.ts audits with axe. Copy from it when building a page. It is managed so the
     audits cannot be deleted by editing it; it is not routed and nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import Button from '../Button.svelte';
	import Field from '../Field.svelte';
	import Table from '../Table.svelte';
	import type { NavGroup } from '../../shell/types';
	import DashboardPage from './DashboardPage.svelte';
	import DetailPage from './DetailPage.svelte';
	import FormPage from './FormPage.svelte';
	import ListPage from './ListPage.svelte';
	import SettingsPage from './SettingsPage.svelte';
	import SettingsSection from './SettingsSection.svelte';
	import SignInPage from './SignInPage.svelte';

	let {
		page,
		status = 'ready',
		rows = [{ name: 'Quarterly summary', owner: 'Finance' }],
		errors = [],
		busy = false,
		signInError
	}: {
		page: 'list' | 'detail' | 'form' | 'dashboard' | 'settings' | 'sign-in';
		status?: 'loading' | 'error' | 'ready';
		rows?: Array<Record<string, string | number>>;
		errors?: Array<{ field?: string; message: string }>;
		busy?: boolean;
		signInError?: string;
	} = $props();

	const nav: NavGroup[] = [
		{ id: null, items: [{ label: 'Home', order: 0, href: '/' }, { label: 'Reports', order: 10, href: '/reports' }] }
	];

	let name = $state('');
	let search = $state('');
	const noop = () => {};
</script>

<AppShell {nav} path="/reports" title="Example app">
	{#if page === 'list'}
		<ListPage
			title="Reports"
			description="Everything your team has written."
			caption="Reports"
			columns={[
				{ key: 'name', label: 'Name' },
				{ key: 'owner', label: 'Owner' }
			]}
			{rows}
			{status}
			onretry={noop}
			emptyDescription="Create the first report to see it here."
		>
			{#snippet actions()}<Button onclick={noop}>New report</Button>{/snippet}
			{#snippet filters()}<Field label="Search reports" type="search" bind:value={search} />{/snippet}
			{#snippet emptyAction()}<Button onclick={noop}>New report</Button>{/snippet}
		</ListPage>
	{:else if page === 'detail'}
		<DetailPage
			title="Quarterly summary"
			back={{ href: '/reports', label: 'All reports' }}
			fields={[
				{ label: 'Owner', value: 'Finance' },
				{ label: 'Status', value: 'Published' },
				{ label: 'Notes', value: '' }
			]}
			{status}
			onretry={noop}
		>
			{#snippet actions()}<Button variant="destructive" onclick={noop}>Delete</Button>{/snippet}
			<h2>Revisions</h2>
			<Table caption="Revisions" columns={[{ key: 'when', label: 'When' }]} rows={[{ when: 'Today' }]} />
		</DetailPage>
	{:else if page === 'form'}
		<FormPage title="New report" submitLabel="Create report" cancelHref="/reports" {errors} {busy} onsubmit={noop}>
			<Field
				id="name"
				label="Name"
				bind:value={name}
				hint="Shown in the list."
				error={errors.length ? 'Enter a name' : undefined}
			/>
			<Field label="Notes" multiline />
		</FormPage>
	{:else if page === 'dashboard'}
		<DashboardPage
			title="Overview"
			stats={[
				{ label: 'Open items', value: 12, hint: '3 more than last week' },
				{ label: 'Completed', value: 48 }
			]}
			{status}
			onretry={noop}
		>
			<h2>Recent activity</h2>
			<Table caption="Recent activity" columns={[{ key: 'what', label: 'What' }]} rows={[{ what: 'Report published' }]} />
		</DashboardPage>
	{:else if page === 'settings'}
		<SettingsPage title="Settings" {errors} {busy} onsubmit={noop}>
			{#snippet sections()}
				<SettingsSection title="Profile" description="How you appear to your team.">
					<Field
						id="display-name"
						label="Display name"
						bind:value={name}
						error={errors.length ? 'Enter a display name' : undefined}
					/>
				</SettingsSection>
				<SettingsSection title="Notifications">
					<fieldset>
						<legend>Email me about</legend>
						<label><input type="checkbox" /> New reports</label>
					</fieldset>
				</SettingsSection>
			{/snippet}
		</SettingsPage>
	{:else}
		<SignInPage error={signInError} resetHref="/reset" signUpHref="/sign-up" onsubmit={noop} />
	{/if}
</AppShell>
