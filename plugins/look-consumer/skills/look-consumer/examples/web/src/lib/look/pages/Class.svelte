<!-- An item with a booking action, in this look: the class facts and a short "about", with a large
     Book a place card beside them (below on a phone). The one big action is the card's button;
     success replaces the form with a friendly confirmation that says what happens next. A starting
     point, then yours: replace the facts and the fields. Built on the kit's ItemPage. -->
<script lang="ts">
	import Field from '../../kit/Field.svelte';
	import Pill from '../../kit/Pill.svelte';
	import ItemPage from '../../kit/pages/ItemPage.svelte';

	let {
		status = 'ready',
		errors = [],
		sent = false
	}: {
		status?: 'loading' | 'error' | 'ready';
		errors?: Array<{ field?: string; message: string }>;
		sent?: boolean;
	} = $props();

	let name = $state('');
	let email = $state('');
	const noop = () => {};
</script>

<ItemPage
	title="Pottery for beginners"
	description="Six relaxed weeks at the wheel."
	back={{ href: '/classes', label: 'All classes' }}
	fields={[
		{ label: 'When', value: 'Tuesdays, 7pm, from 2027-01-12' },
		{ label: 'Where', value: 'Town hall, room 2' },
		{ label: 'Price', value: '£48 for six weeks' }
	]}
	{status}
	onretry={noop}
	enquiryTitle="Book a place"
	submitLabel="Book a place"
	{errors}
	{sent}
	sentTitle="You are booked in"
	sentMessage="We have emailed your booking. Bring an apron."
	onsubmit={noop}
>
	{#snippet badge()}<Pill tone="success">Places left</Pill>{/snippet}
	{#snippet about()}
		<h2>About the class</h2>
		<p>No experience needed. We cover the basics of centring, pulling up and trimming.</p>
	{/snippet}
	{#snippet enquiry()}
		<Field id="name" label="Your name" bind:value={name} error={errors.find((e) => e.field === 'name')?.message} />
		<Field
			id="email"
			label="Email"
			type="email"
			bind:value={email}
			error={errors.find((e) => e.field === 'email')?.message}
		/>
	{/snippet}
</ItemPage>
