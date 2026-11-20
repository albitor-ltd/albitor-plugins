<!-- An inner page of a portal, in this look: the link back to the hub first, then a dense table with
     exact dates. The kit's ListPage has no back link of its own, so the page opens with BackToHub.
     A starting point, then yours: replace the columns and rows. Built on BackToHub and ListPage. -->
<script lang="ts">
	import FileText from '@lucide/svelte/icons/file-text';
	import BackToHub from '../../kit/pages/BackToHub.svelte';
	import ListPage from '../../kit/pages/ListPage.svelte';

	let {
		status = 'ready',
		rows = [
			{ sent: '2026-11-01', subject: 'Your permit expires soon', ref: 'LTR-3321' },
			{ sent: '2026-06-02', subject: 'Payment received', ref: 'LTR-2907' }
		]
	}: {
		status?: 'loading' | 'error' | 'ready';
		rows?: Array<Record<string, string | number>>;
	} = $props();
	const noop = () => {};
</script>

<BackToHub href="/" label="Back to your permits" />

<ListPage
	title="Letters"
	description="Letters we have sent you in the last 12 months."
	caption="Letters"
	columns={[
		{ key: 'sent', label: 'Sent' },
		{ key: 'subject', label: 'Subject' },
		{ key: 'ref', label: 'Reference', priority: 'secondary' }
	]}
	{rows}
	{status}
	onretry={noop}
	emptyTitle="No letters yet"
	emptyDescription="Letters about your permits appear here when we send them."
	emptyIcon={FileText}
/>
