<!-- The first screen of a portal, in this look: plain bordered tiles of exact status, no shadow and
     no decoration, then a compact table of what is open with figures in tabular type. There is no
     bar of destinations (pass `nav={[]}` to AppShell): the tiles are the navigation, and every
     other page keeps a link back with BackToHub. A starting point, then yours: replace the tiles
     and the table. Built on the kit's HubPage and Table. -->
<script lang="ts">
	import Banknote from '@lucide/svelte/icons/banknote';
	import FileText from '@lucide/svelte/icons/file-text';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import Table from '../../kit/Table.svelte';
	import HubPage from '../../kit/pages/HubPage.svelte';

	let { status = 'ready' }: { status?: 'loading' | 'error' | 'ready' } = $props();
	const noop = () => {};
</script>

<HubPage
	title="Your parking permits"
	description="Renew, pay and read letters about your permits."
	tiles={[
		{
			href: '/renew',
			title: 'Renew a permit',
			text: 'Permit PRM-48213 expires on 2026-12-01.',
			icon: RefreshCw,
			status: { tone: 'warning', label: 'Due 2026-12-01' }
		},
		{
			href: '/payments',
			title: 'Payments',
			text: 'Last payment £120.00 on 2026-06-02.',
			icon: Banknote,
			status: { tone: 'success', label: 'Paid up' }
		},
		{ href: '/letters', title: 'Letters', text: '2 letters in the last 12 months.', icon: FileText }
	]}
	{status}
	onretry={noop}
>
	<h2>Open items</h2>
	<div class="figures">
		<Table
			caption="Open items"
			columns={[
				{ key: 'ref', label: 'Reference' },
				{ key: 'item', label: 'Item' },
				{ key: 'due', label: 'Due' },
				{ key: 'amount', label: 'Amount' }
			]}
			rows={[{ ref: 'PRM-48213', item: 'Annual renewal', due: '2026-12-01', amount: '£120.00' }]}
		/>
	</div>
</HubPage>

<style>
	.figures {
		font-variant-numeric: var(--figures, tabular-nums);
	}
</style>
