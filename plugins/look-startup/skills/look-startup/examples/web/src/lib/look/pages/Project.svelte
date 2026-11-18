<!-- A detail page, in this look: the status Pill beside the title, one primary action and the rest
     quiet, the facts as a description list, then tabs for the related views and a sparkline for
     the trend. A starting point, then yours: replace the facts, the tabs and the actions. Needs
     the kit's DetailPage (copy it from the kit-shadcn skill). A destructive action opens a Dialog;
     never delete on one click. -->
<script lang="ts">
	import Button from '../../kit/Button.svelte';
	import Card from '../../kit/Card.svelte';
	import Chart from '../../kit/Chart.svelte';
	import Pill from '../../kit/Pill.svelte';
	import Tabs from '../../kit/Tabs.svelte';
	import Table from '../../kit/Table.svelte';
	import DetailPage from '../../kit/pages/DetailPage.svelte';

	let { status = 'ready' }: { status?: 'loading' | 'error' | 'ready' } = $props();

	let tab = $state('history');
	const buildTimes = [
		{ label: 'Mon', value: 74 },
		{ label: 'Tue', value: 71 },
		{ label: 'Wed', value: 69 },
		{ label: 'Thu', value: 64 },
		{ label: 'Fri', value: 58 }
	];
	const noop = () => {};
</script>

<DetailPage
	title="api-gateway"
	description="Public API for the web and mobile apps."
	back={{ href: '/projects', label: 'All projects' }}
	fields={[
		{ label: 'Team', value: 'Platform' },
		{ label: 'Branch', value: 'main' },
		{ label: 'Last deployed', value: '12 minutes ago' },
		{ label: 'Region', value: 'eu-west-2' }
	]}
	{status}
	onretry={noop}
>
	{#snippet badge()}<Pill tone="success">Live</Pill>{/snippet}
	{#snippet actions()}
		<Button variant="secondary" onclick={noop}>Roll back</Button>
		<Button onclick={noop}>Deploy now</Button>
	{/snippet}
	<Card title="Build time">
		<Chart kind="sparkline" title="Build time in seconds, Monday to Friday" data={buildTimes} unit=" s" />
	</Card>
	<Tabs
		label="Project views"
		tabs={[
			{ value: 'history', label: 'History' },
			{ value: 'settings', label: 'Configuration' }
		]}
		bind:value={tab}
	>
		{#snippet panel(value)}
			{#if value === 'history'}
				<Table
					caption="Deployment history"
					columns={[
						{ key: 'id', label: 'Deployment' },
						{ key: 'when', label: 'When' },
						{ key: 'who', label: 'By' }
					]}
					rows={[
						{ id: '#412', when: '12 minutes ago', who: 'Ana' },
						{ id: '#411', when: 'Yesterday', who: 'Sam' }
					]}
				/>
			{:else}
				<p>Build command, environment variables and domains are set per project.</p>
			{/if}
		{/snippet}
	</Tabs>
</DetailPage>
