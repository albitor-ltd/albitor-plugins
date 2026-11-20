<!-- Test support: ItemPage in the shell, for itempage.test.ts alone. Delete it with ItemPage.svelte
     and itempage.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import Field from '../../Field.svelte';
	import Pill from '../../Pill.svelte';
	import ItemPage from '../ItemPage.svelte';
	import PagesShowcase from './PagesShowcase.svelte';

	let {
		status = 'ready',
		errors = [],
		busy = false,
		sent: initiallySent = false
	}: {
		status?: 'loading' | 'error' | 'ready';
		errors?: Array<{ field?: string; message: string }>;
		busy?: boolean;
		sent?: boolean;
	} = $props();

	// Writable derived: a test flips it with send(), as a submit that succeeds does.
	let sent = $derived(initiallySent);
	export function send() {
		sent = true;
	}

	let email = $state('');
	let message = $state('');
	const noop = () => {};
</script>

<PagesShowcase>
	<ItemPage
		title="Pottery for beginners"
		description="Six weeks at the wheel."
		back={{ href: '/courses', label: 'All courses' }}
		fields={[
			{ label: 'When', value: 'Tuesdays, 7pm' },
			{ label: 'Where', value: 'Hall 2' }
		]}
		{status}
		onretry={noop}
		enquiryTitle="Book a place"
		submitLabel="Book a place"
		{errors}
		{busy}
		{sent}
		sentMessage="We have emailed your booking."
		onsubmit={noop}
	>
		{#snippet badge()}<Pill tone="success">Places left</Pill>{/snippet}
		{#snippet about()}
			<h2>About the course</h2>
			<p>No experience needed.</p>
		{/snippet}
		{#snippet enquiry()}
			<Field
				id="email"
				label="Email"
				type="email"
				bind:value={email}
				error={errors.length ? 'Enter an email address' : undefined}
			/>
			<Field label="Message" multiline bind:value={message} />
		{/snippet}
	</ItemPage>
</PagesShowcase>
