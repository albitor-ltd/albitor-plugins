<!-- A detail page, in this look: the status Pill beside the title, a progress ring in a roomy
     card, the facts as a description list and the one main action. A starting point, then yours:
     replace the facts, the progress and the actions. Needs the kit's DetailPage (copy it from the
     kit-shadcn skill). A destructive action opens a Dialog; never delete on one click. -->
<script lang="ts">
	import Button from '../../kit/Button.svelte';
	import Card from '../../kit/Card.svelte';
	import Dialog from '../../kit/Dialog.svelte';
	import Pill from '../../kit/Pill.svelte';
	import Progress from '../../kit/Progress.svelte';
	import DetailPage from '../../kit/pages/DetailPage.svelte';

	let { status = 'ready' }: { status?: 'loading' | 'error' | 'ready' } = $props();

	let confirming = $state(false);
	const noop = () => {};
</script>

<DetailPage
	title="Evening walk"
	description="Ten minutes outside, after dinner."
	back={{ href: '/habits', label: 'All habits' }}
	fields={[
		{ label: 'How often', value: 'Every day' },
		{ label: 'Reminder', value: '19:30' },
		{ label: 'Started', value: '2026-10-28' },
		{ label: 'Best streak', value: '31 days' }
	]}
	{status}
	onretry={noop}
>
	{#snippet badge()}<Pill tone="warning">To do today</Pill>{/snippet}
	{#snippet actions()}
		<Button variant="ghost" onclick={() => (confirming = true)}>Delete habit</Button>
		<Button onclick={noop}>Mark done</Button>
	{/snippet}
	<Card title="This week">
		<Progress label="Days done this week" variant="ring" value={5} max={7} showValue />
	</Card>
</DetailPage>

<Dialog
	bind:open={confirming}
	title="Delete this habit?"
	description="Evening walk and its streak are removed. This cannot be undone."
>
	{#snippet actions()}
		<Button variant="secondary" onclick={() => (confirming = false)}>Keep habit</Button>
		<Button variant="destructive" onclick={() => (confirming = false)}>Delete habit</Button>
	{/snippet}
</Dialog>
