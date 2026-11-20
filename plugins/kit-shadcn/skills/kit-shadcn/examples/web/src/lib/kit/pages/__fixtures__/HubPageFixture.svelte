<!-- Test support: HubPage and BackToHub in an AppShell with no nav (the hub is the navigation), for
     hubpage.test.ts alone. Delete it with HubPage.svelte, BackToHub.svelte and hubpage.test.ts.
     Not routed; nothing links here. -->
<script lang="ts">
	import FileText from '@lucide/svelte/icons/file-text';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import AppShell from '../../AppShell.svelte';
	import BackToHub from '../BackToHub.svelte';
	import HubPage from '../HubPage.svelte';
	import PageHeader from '../PageHeader.svelte';

	let {
		page = 'hub',
		status = 'ready'
	}: { page?: 'hub' | 'inner' | 'inner-header'; status?: 'loading' | 'error' | 'ready' } = $props();
	const noop = () => {};
</script>

<AppShell nav={[]} path={page === 'hub' ? '/' : '/renew'} title="Example app">
	{#if page === 'hub'}
		<HubPage
			title="Your account"
			description="Everything you can do, in one place."
			tiles={[
				{
					href: '/renew',
					title: 'Renew your permit',
					text: 'Takes about 5 minutes.',
					icon: RefreshCw,
					status: { tone: 'warning', label: 'Due 1 December' }
				},
				{ href: '/letters', title: 'Your letters', text: 'Everything we have sent you.', icon: FileText }
			]}
			{status}
			onretry={noop}
		>
			<h2>Recent activity</h2>
			<p>Permit issued on 1 December 2025.</p>
		</HubPage>
	{:else if page === 'inner'}
		<BackToHub href="/" label="Back to your account" />
		<h1>Renew your permit</h1>
	{:else}
		<PageHeader title="Your letters" back={{ href: '/', label: 'Back to your account' }} />
	{/if}
</AppShell>
