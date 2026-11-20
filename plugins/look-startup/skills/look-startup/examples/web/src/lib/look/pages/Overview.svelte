<!-- The dashboard first screen, in this look. Use it only when the app's structure names a
     dashboard (a summary of data that changes without the person's own input); for any other
     structure start from the non-dashboard pages beside it (Home, Release, Trial). The overview: layered tiles with an icon, a change in words, a meter or a
     sparkline, then two cards of detail. A starting point, then yours: replace the figures and
     lists with the app's own, and pass its load `status`. Needs the kit's DashboardPage (copy it
     from the kit-shadcn skill). Before the app has any data, pass `firstRun`. -->
<script lang="ts">
	import Activity from '@lucide/svelte/icons/activity';
	import FolderKanban from '@lucide/svelte/icons/folder-kanban';
	import Plus from '@lucide/svelte/icons/plus';
	import Rocket from '@lucide/svelte/icons/rocket';
	import Timer from '@lucide/svelte/icons/timer';
	import Icon from '../../kit/Icon.svelte';
	import Button from '../../kit/Button.svelte';
	import Card from '../../kit/Card.svelte';
	import Chart from '../../kit/Chart.svelte';
	import Pill from '../../kit/Pill.svelte';
	import DashboardPage from '../../kit/pages/DashboardPage.svelte';

	let {
		status = 'ready',
		firstRun = false
	}: { status?: 'loading' | 'error' | 'ready'; firstRun?: boolean } = $props();

	const week = [
		{ label: 'Mon', value: 6 },
		{ label: 'Tue', value: 9 },
		{ label: 'Wed', value: 7 },
		{ label: 'Thu', value: 12 },
		{ label: 'Fri', value: 10 }
	];
	const buildTimes = [
		{ label: 'Mon', value: 74 },
		{ label: 'Tue', value: 71 },
		{ label: 'Wed', value: 69 },
		{ label: 'Thu', value: 64 },
		{ label: 'Fri', value: 58 }
	];
	const recent = [
		{ name: 'api-gateway', when: '12 minutes ago', tone: 'success', state: 'Live' },
		{ name: 'billing-worker', when: '1 hour ago', tone: 'danger', state: 'Failed' },
		{ name: 'web-app', when: '3 hours ago', tone: 'warning', state: 'Rolling out' }
	] as const;
	const noop = () => {};
</script>

<DashboardPage
	title="Overview"
	description="Deployments and build health for this week."
	stats={[
		{
			label: 'Deployments',
			value: 44,
			icon: Rocket,
			delta: { direction: 'up', text: '8 more than last week', good: true }
		},
		{ label: 'Success rate', value: '96%', icon: Activity, meter: { value: 96, label: 'Share that succeeded' } },
		{
			label: 'Median build time',
			value: '58 s',
			icon: Timer,
			spark: { data: buildTimes, label: 'Median build time, Monday to Friday' }
		},
		{ label: 'Projects', value: 12, icon: FolderKanban }
	]}
	{firstRun}
	firstRunTitle="No projects yet"
	firstRunDescription="Create a project and its deployments and build times appear here."
	firstRunIcon={FolderKanban}
	{status}
	onretry={noop}
>
	{#snippet actions()}
		<Button onclick={noop}><Icon icon={Plus} size={16} /> New project</Button>
	{/snippet}
	{#snippet firstRunAction()}
		<Button onclick={noop}>Create a project</Button>
	{/snippet}
	<div class="two">
		<Card title="Deployments this week">
			<Chart title="Deployments per day" data={week} />
		</Card>
		<Card title="Recent deployments" tone="second">
			<ul class="recent">
				{#each recent as item (item.name)}
					<li>
						<span class="what"><b>{item.name}</b><small>{item.when}</small></span>
						<Pill tone={item.tone}>{item.state}</Pill>
					</li>
				{/each}
			</ul>
		</Card>
	</div>
</DashboardPage>

<style>
	.two {
		display: grid;
		grid-template-columns: minmax(0, 1.6fr) minmax(0, 1fr);
		gap: var(--space-4);
	}
	@media (width < 48rem) {
		.two {
			grid-template-columns: 1fr;
		}
	}
	.recent {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.recent li {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3) 0;
		border-top: var(--border-width) solid var(--border);
	}
	.recent li:first-child {
		border-top: 0;
	}
	.what {
		display: grid;
	}
	.what small {
		color: var(--muted-foreground);
		font-size: var(--text-small);
	}
</style>
