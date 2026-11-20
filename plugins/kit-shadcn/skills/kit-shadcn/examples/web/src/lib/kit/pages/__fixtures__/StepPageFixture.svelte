<!-- Test support: StepPage and ConfirmationPage in the shell, for steppage.test.ts alone. Delete it
     with StepPage.svelte, ConfirmationPage.svelte and steppage.test.ts. Not routed; nothing links
     here. -->
<script lang="ts">
	import Field from '../../Field.svelte';
	import ConfirmationPage from '../ConfirmationPage.svelte';
	import StepPage from '../StepPage.svelte';
	import PagesShowcase from './PagesShowcase.svelte';

	let {
		phase = 'question',
		step = 2,
		errors = [],
		busy = false
	}: {
		phase?: 'question' | 'check' | 'done';
		step?: number;
		errors?: Array<{ field?: string; message: string }>;
		busy?: boolean;
	} = $props();

	let postcode = $state('');
	const noop = () => {};
</script>

<PagesShowcase>
	{#if phase === 'question'}
		<StepPage
			title="What is your postcode?"
			description="We use it to find your council."
			flow="Renew a permit"
			{step}
			total={4}
			backHref={step > 1 ? '/renew/1' : undefined}
			{errors}
			{busy}
			onsubmit={noop}
		>
			<Field
				id="postcode"
				label="Postcode"
				bind:value={postcode}
				error={errors.length ? 'Enter a postcode' : undefined}
			/>
		</StepPage>
	{:else if phase === 'check'}
		<StepPage
			title="Check your answers"
			flow="Renew a permit"
			step={4}
			total={4}
			backHref="/renew/3"
			submitLabel="Submit renewal"
			answers={[
				{ label: 'Postcode', value: 'AB1 2CD', changeHref: '/renew/2' },
				{ label: 'Vehicle', value: '', changeHref: '/renew/3' }
			]}
			onsubmit={noop}
		/>
	{:else}
		<ConfirmationPage
			title="Renewal sent"
			reference="PRM-48213"
			next={['We email you within 3 working days.', 'Your current permit stays valid until then.']}
			action={{ href: '/', label: 'Back to your account' }}
		/>
	{/if}
</PagesShowcase>
