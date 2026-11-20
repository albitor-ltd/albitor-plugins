<!-- One step of a linear flow (a wizard): one thing per page. "Step 2 of 5", a Progress bar that says
     where the person is, the question as the <h1>, its fields, and Back and Continue. Each step is
     a real page (its own route, or one route with the step in the URL) so the browser's Back works
     and a refresh keeps the place. `backHref` is the previous step; the first step has none (or a
     link out, such as BackToHub). Pass `errors` on a failed Continue; the summary takes focus.

     The two pages that close a flow are modes of this one: pass `answers` for "Check your answers"
     (each answer with a Change link, and the last button names the action, "Submit application"),
     and use ConfirmationPage for the page after it.

       <StepPage title="What is your postcode?" flow="Renew a permit" step={2} total={5}
                 backHref="/renew/1" {errors} onsubmit={next}>
         <Field id="postcode" label="Postcode" bind:value={postcode} />
       </StepPage> -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Button from '../Button.svelte';
	import Progress from '../Progress.svelte';
	import ActionLink from './ActionLink.svelte';
	import ErrorSummary from './ErrorSummary.svelte';

	let {
		title,
		description,
		flow,
		step,
		total,
		backHref,
		submitLabel = 'Continue',
		answers,
		errors = [],
		busy = false,
		onsubmit,
		children
	}: {
		/** The question this step asks, or "Check your answers". */
		title: string;
		description?: string;
		/** The name of the whole flow, for the progress bar's label. */
		flow: string;
		step: number;
		total: number;
		backHref?: string;
		/** Name the action on the last step ("Submit application"), not "Submit". */
		submitLabel?: string;
		/** Check-answers mode: each answer with where to change it. */
		answers?: Array<{ label: string; value: string; changeHref: string }>;
		errors?: Array<{ field?: string; message: string }>;
		busy?: boolean;
		onsubmit: () => void | Promise<void>;
		/** The step's labelled Fields; leave out in check-answers mode. */
		children?: Snippet;
	} = $props();

	function handle(event: SubmitEvent) {
		event.preventDefault();
		if (!busy) void onsubmit();
	}
</script>

<div class="step">
	<p class="where">Step {step} of {total}</p>
	<Progress label={`${flow} progress`} value={step} max={total} />
	<h1>{title}</h1>
	{#if description}<p class="description">{description}</p>{/if}

	<ErrorSummary {errors} />

	<form onsubmit={handle} novalidate>
		{#if answers}
			<dl>
				{#each answers as answer (answer.label)}
					<div>
						<dt>{answer.label}</dt>
						<dd>{answer.value === '' ? 'Not answered' : answer.value}</dd>
						<dd class="change"><a href={answer.changeHref}>Change{' '}<span class="sr">{answer.label}</span></a></dd>
					</div>
				{/each}
			</dl>
		{:else if children}
			<div class="fields">{@render children()}</div>
		{/if}
		<div class="foot">
			{#if backHref}<ActionLink href={backHref} variant="secondary">Back</ActionLink>{/if}
			<Button type="submit" disabled={busy}>{busy ? 'Saving…' : submitLabel}</Button>
		</div>
	</form>
</div>

<style>
	.step {
		max-width: 40rem;
	}
	.where {
		margin: 0 0 var(--space-2);
		color: var(--muted-foreground);
		font-size: var(--text-small);
	}
	h1 {
		margin: var(--space-6) 0 var(--space-2);
		font-size: var(--text-title);
	}
	.description {
		margin: 0 0 var(--space-4);
		color: var(--muted-foreground);
	}
	.fields {
		display: grid;
		gap: var(--space-4);
		margin: var(--space-4) 0;
	}
	dl {
		margin: var(--space-4) 0;
		border-top: var(--border-width) solid var(--border);
	}
	dl div {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 2fr) auto;
		gap: var(--space-3);
		align-items: center;
		padding: var(--space-3) 0;
		border-bottom: var(--border-width) solid var(--border);
	}
	@media (width < 40rem) {
		dl div {
			grid-template-columns: 1fr auto;
		}
		dt {
			grid-column: 1 / -1;
		}
	}
	dt {
		color: var(--muted-foreground);
	}
	dd {
		margin: 0;
		overflow-wrap: anywhere;
	}
	.change a {
		display: inline-flex;
		align-items: center;
		min-height: 2.75rem;
		color: var(--link);
	}
	.sr {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}
	.foot {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		margin-top: var(--space-6);
	}
</style>
