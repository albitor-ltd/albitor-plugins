<!-- A step flow with its own way back, in this look: the question page, Check your answers and the
     confirmation, each opening with the link back to the hub. Dense and plain: exact answers in a
     ruled list, one named action per page, the reference in tabular type. Every step is a real
     page of the app's routes. A starting point, then yours: replace the questions. Built on the
     kit's StepPage, ConfirmationPage and BackToHub. -->
<script lang="ts">
	import Field from '../../kit/Field.svelte';
	import BackToHub from '../../kit/pages/BackToHub.svelte';
	import ConfirmationPage from '../../kit/pages/ConfirmationPage.svelte';
	import StepPage from '../../kit/pages/StepPage.svelte';

	let {
		phase = 'question',
		errors = []
	}: {
		phase?: 'question' | 'check' | 'done';
		errors?: Array<{ field?: string; message: string }>;
	} = $props();

	let registration = $state('');
	const noop = () => {};
</script>

<BackToHub href="/" label="Back to your permits" />

{#if phase === 'question'}
	<StepPage
		title="What is the vehicle registration?"
		description="As it appears on your V5C."
		flow="Renew a permit"
		step={1}
		total={2}
		{errors}
		onsubmit={noop}
	>
		<Field
			id="registration"
			label="Registration number"
			bind:value={registration}
			error={errors[0]?.message}
		/>
	</StepPage>
{:else if phase === 'check'}
	<StepPage
		title="Check your answers"
		flow="Renew a permit"
		step={2}
		total={2}
		backHref="/renew"
		submitLabel="Renew the permit"
		answers={[
			{ label: 'Permit', value: 'PRM-48213', changeHref: '/renew' },
			{ label: 'Registration number', value: 'AB12 CDE', changeHref: '/renew' },
			{ label: 'Fee', value: '£120.00', changeHref: '/renew' }
		]}
		onsubmit={noop}
	/>
{:else}
	<ConfirmationPage
		title="Permit renewed"
		reference="PRM-48213"
		next={['A confirmation letter is sent within 3 working days.', 'Your new permit runs from 2026-12-01.']}
		action={{ href: '/', label: 'Back to your permits' }}
	/>
{/if}
