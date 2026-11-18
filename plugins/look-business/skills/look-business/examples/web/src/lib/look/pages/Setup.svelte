<!-- A step flow, in this look: the steps listed in order with the current one marked, the fields
     for that step, Back and Continue at the foot. Each step is a real page in the app's own
     routes (or one route with a step in the URL), so Back works. A starting point, then yours:
     replace the steps and the fields. Built from PageHeader (copy the kit's page templates from
     the kit-shadcn skill). -->
<script lang="ts">
	import Button from '../../kit/Button.svelte';
	import Field from '../../kit/Field.svelte';
	import PageHeader from '../../kit/pages/PageHeader.svelte';

	let { step = 1 }: { step?: 1 | 2 | 3 } = $props();

	const steps = ['Supplier details', 'Payment terms', 'Check and create'] as const;
	let name = $state('');
	let terms = $state('30');
	const noop = () => {};
</script>

<PageHeader title="Add a supplier" description={`Step ${step} of ${steps.length}`} />

<ol class="steps" aria-label="Steps">
	{#each steps as label, i (label)}
		<li aria-current={i + 1 === step ? 'step' : undefined} class:done={i + 1 < step}>
			<span class="n">{i + 1}</span>
			{label}{#if i + 1 < step}<span class="sr"> (done)</span>{/if}
		</li>
	{/each}
</ol>

<form
	onsubmit={(e) => {
		e.preventDefault();
		noop();
	}}
>
	<h2>{steps[step - 1]}</h2>
	{#if step === 1}
		<Field id="name" label="Supplier name" bind:value={name} />
	{:else if step === 2}
		<Field id="terms" label="Payment terms in days" inputmode="numeric" bind:value={terms} />
	{:else}
		<p>Create {name || 'the supplier'} with {terms}-day payment terms.</p>
	{/if}
	<div class="foot">
		{#if step > 1}<Button variant="secondary" onclick={noop}>Back</Button>{:else}<span></span>{/if}
		<Button type="submit">{step === 3 ? 'Create supplier' : 'Continue'}</Button>
	</div>
</form>

<style>
	.steps {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-4);
		list-style: none;
		margin: 0 0 var(--space-6);
		padding: 0;
		color: var(--muted-foreground);
	}
	.steps li {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}
	.steps li[aria-current='step'] {
		color: var(--foreground);
		font-weight: 600;
	}
	.n {
		display: inline-grid;
		place-items: center;
		inline-size: 1.5rem;
		block-size: 1.5rem;
		border: var(--border-width) solid currentColor;
		border-radius: var(--radius);
		font-variant-numeric: var(--figures);
	}
	li[aria-current='step'] .n {
		background: var(--primary);
		color: var(--primary-foreground);
		border-color: var(--primary);
	}
	.sr {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}
	form {
		display: grid;
		gap: var(--space-4);
		max-width: 36rem;
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
