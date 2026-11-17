<!-- Step 2: one thing per page. The question is the page's one h1 (the label is the heading), with
     a Back link above it, optional hint text, and one primary button that names the way on
     ("Continue"). A failed submit shows the error summary above the heading and the same message
     beside the control. Validate on the server too: the browser is not a boundary.

       <QuestionPage question="What is your name?" id="name" bind:value={name}
                     backHref="/apply" error={error} onsubmit={next} /> -->
<script lang="ts">
	import BackLink from '../BackLink.svelte';
	import Button from '../Button.svelte';
	import ErrorSummary from '../ErrorSummary.svelte';

	let {
		question,
		id,
		value = $bindable(''),
		hint,
		error,
		backHref,
		submitLabel = 'Continue',
		onsubmit
	}: {
		/** The question, as the page heading: "What is your name?" */
		question: string;
		/** The control's id; the error summary links to it. */
		id: string;
		value?: string;
		hint?: string;
		/** One plain sentence saying what to do: "Enter your full name". */
		error?: string;
		/** The page the person came from. */
		backHref: string;
		submitLabel?: string;
		onsubmit: () => void | Promise<void>;
	} = $props();

	const hintId = $derived(`${id}-hint`);
	const errorId = $derived(`${id}-error`);
	const describedBy = $derived([hint ? hintId : '', error ? errorId : ''].filter(Boolean).join(' '));

	function handle(event: SubmitEvent) {
		event.preventDefault();
		void onsubmit();
	}
</script>

<BackLink href={backHref} />

<div class="govuk-grid-row">
	<div class="govuk-grid-column-two-thirds">
		{#if error}<ErrorSummary errors={[{ field: id, message: error }]} />{/if}

		<form onsubmit={handle} novalidate>
			<div class="govuk-form-group" class:govuk-form-group--error={error}>
				<h1 class="govuk-label-wrapper">
					<label class="govuk-label govuk-label--l" for={id}>{question}</label>
				</h1>
				{#if hint}<div class="govuk-hint" id={hintId}>{hint}</div>{/if}
				{#if error}
					<p class="govuk-error-message" id={errorId}>
						<span class="govuk-visually-hidden">Error:</span>
						{error}
					</p>
				{/if}
				<input
					class="govuk-input"
					class:govuk-input--error={error}
					{id}
					name={id}
					type="text"
					bind:value
					aria-describedby={describedBy || undefined}
					aria-invalid={error ? 'true' : undefined}
				/>
			</div>
			<Button type="submit">{submitLabel}</Button>
		</form>
	</div>
</div>
