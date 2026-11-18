<!-- A step flow, in this look: one question at a time, big tappable choices, a Progress bar that
     says where the person is, and one main action at the foot. Each step is a real page in the
     app's own routes (or one route with a step in the URL), so Back works. A starting point,
     then yours: replace the questions and the choices. Built from PageHeader (copy the kit's page
     templates from the kit-shadcn skill). -->
<script lang="ts">
	import Button from '../../kit/Button.svelte';
	import Progress from '../../kit/Progress.svelte';
	import PageHeader from '../../kit/pages/PageHeader.svelte';

	let { step = 1 }: { step?: 1 | 2 | 3 } = $props();

	const questions = ['What do you want to keep up?', 'How often?', 'When should we remind you?'] as const;
	const choices: Array<string[]> = [
		['A walk', 'Reading', 'Stretching'],
		['Every day', 'Weekdays', 'Twice a week'],
		['Morning', 'Lunchtime', 'Evening']
	];
	let picked = $state<string[]>(['', '', '']);
	const noop = () => {};
</script>

<PageHeader title="Add a habit" description={`Question ${step} of 3`} />

<div class="progress">
	<Progress label="Questions answered" value={step} max={3} />
</div>

<form
	onsubmit={(e) => {
		e.preventDefault();
		noop();
	}}
>
	<fieldset>
		<legend>{questions[step - 1]}</legend>
		{#each choices[step - 1] as choice (choice)}
			<label class="opt" class:on={picked[step - 1] === choice}>
				<input type="radio" name={`step-${step}`} value={choice} bind:group={picked[step - 1]} />
				{choice}
			</label>
		{/each}
	</fieldset>
	<div class="foot">
		{#if step > 1}<Button variant="ghost" onclick={noop}>Back</Button>{:else}<span></span>{/if}
		<Button type="submit">{step === 3 ? 'Add habit' : 'Next'}</Button>
	</div>
</form>

<style>
	.progress {
		margin-bottom: var(--space-6);
		max-width: 36rem;
	}
	form {
		display: grid;
		gap: var(--space-6);
		max-width: 36rem;
	}
	fieldset {
		border: 0;
		margin: 0;
		padding: 0;
		display: grid;
		gap: var(--space-3);
	}
	legend {
		font-family: var(--heading-font);
		font-weight: var(--heading-weight);
		font-size: var(--text-heading);
		margin-bottom: var(--space-3);
	}
	.opt {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		min-height: 3rem;
		padding: var(--space-3) var(--space-4);
		border: var(--border-width) solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		cursor: pointer;
	}
	.opt.on {
		border-color: var(--primary);
		box-shadow: 0 0 0 1px var(--primary);
	}
	.opt:has(input:focus-visible) {
		outline: var(--focus-width) solid var(--ring);
		outline-offset: 2px;
	}
	.foot {
		display: flex;
		justify-content: space-between;
		gap: var(--space-3);
	}
</style>
