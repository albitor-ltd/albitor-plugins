<!-- A step flow, in this look: one question group per step, a labelled Progress bar that says where
     the person is, Back and Continue at the foot. Each step is a real page in the app's own
     routes (or one route with a step in the URL), so Back works. A starting point, then yours:
     replace the steps and the fields. Built from PageHeader (copy the kit's page templates from
     the kit-shadcn skill). -->
<script lang="ts">
	import Button from '../../kit/Button.svelte';
	import Field from '../../kit/Field.svelte';
	import Progress from '../../kit/Progress.svelte';
	import PageHeader from '../../kit/pages/PageHeader.svelte';

	let { step = 1 }: { step?: 1 | 2 | 3 } = $props();

	const titles = ['Name the project', 'Connect the repository', 'Choose a region'] as const;
	let name = $state('');
	let repo = $state('');
	let region = $state('eu-west-2');
	const noop = () => {};
</script>

<PageHeader title="Set up a project" description={`Step ${step} of 3: ${titles[step - 1]}`} />

<div class="progress">
	<Progress label="Setup progress" value={step} max={3} />
</div>

<form
	onsubmit={(e) => {
		e.preventDefault();
		noop();
	}}
>
	<h2>{titles[step - 1]}</h2>
	{#if step === 1}
		<Field id="name" label="Project name" hint="Lowercase letters, numbers and hyphens." bind:value={name} />
	{:else if step === 2}
		<Field id="repo" label="Repository" hint="For example example/api-gateway." bind:value={repo} />
	{:else}
		<Field id="region" label="Region" bind:value={region} />
	{/if}
	<div class="foot">
		{#if step > 1}<Button variant="secondary" onclick={noop}>Back</Button>{:else}<span></span>{/if}
		<Button type="submit">{step === 3 ? 'Create project' : 'Continue'}</Button>
	</div>
</form>

<style>
	.progress {
		margin-bottom: var(--space-6);
		max-width: 40rem;
	}
	form {
		display: grid;
		gap: var(--space-4);
		max-width: 40rem;
	}
	h2 {
		margin: 0;
		font-size: var(--text-heading);
	}
	.foot {
		display: flex;
		justify-content: space-between;
		gap: var(--space-3);
	}
</style>
