<!-- A step flow that ends in a confirmation, in this look: one question per page, a labelled bar
     that says where the person is, Back and Continue, then Check your answers and a confirmation
     with a reference. One primary action per page. Each step is a real page of the app's own
     routes. A starting point, then yours: replace the questions. Built on the kit's StepPage and
     ConfirmationPage. -->
<script lang="ts">
	import Field from '../../kit/Field.svelte';
	import ConfirmationPage from '../../kit/pages/ConfirmationPage.svelte';
	import StepPage from '../../kit/pages/StepPage.svelte';

	let {
		phase = 'question',
		step = 1,
		errors = []
	}: {
		phase?: 'question' | 'check' | 'done';
		step?: 1 | 2;
		errors?: Array<{ field?: string; message: string }>;
	} = $props();

	let email = $state('');
	let repo = $state('');
	const noop = () => {};
</script>

{#if phase === 'question'}
	<StepPage
		title={step === 1 ? 'What is your work email?' : 'Which repository should we connect?'}
		description={step === 1 ? 'We send the trial details here.' : 'You can change this later.'}
		flow="Start a trial"
		{step}
		total={3}
		backHref={step === 2 ? '/trial/1' : undefined}
		{errors}
		onsubmit={noop}
	>
		{#if step === 1}
			<Field id="email" label="Work email" type="email" bind:value={email} error={errors[0]?.message} />
		{:else}
			<Field id="repo" label="Repository" hint="For example example/api-gateway." bind:value={repo} />
		{/if}
	</StepPage>
{:else if phase === 'check'}
	<StepPage
		title="Check your answers"
		flow="Start a trial"
		step={3}
		total={3}
		backHref="/trial/2"
		submitLabel="Start the trial"
		answers={[
			{ label: 'Work email', value: 'sam@example.com', changeHref: '/trial/1' },
			{ label: 'Repository', value: 'example/api-gateway', changeHref: '/trial/2' }
		]}
		onsubmit={noop}
	/>
{:else}
	<ConfirmationPage
		title="Your trial has started"
		reference="TRL-20418"
		next={['Check your email for the sign-in link.', 'The first build starts when you push.']}
		action={{ href: '/projects', label: 'Go to your projects' }}
	/>
{/if}
