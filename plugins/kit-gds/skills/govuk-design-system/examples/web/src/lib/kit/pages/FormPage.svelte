<!-- Create or edit one record. Put labelled Fields in the children. On a failed submit, pass
     `errors` as `{ field: 'name', message: 'Enter a name' }` (field is the Field's `id`, so the
     summary links to the control; WCAG 2.4.3): the GOV.UK error summary appears above the heading
     and takes focus so a screen reader announces what to fix (WCAG 3.3.1), and each Field also
     gets its own `error` next to the control, with the same text. Validate on the server too: the
     browser is not a boundary.

       <FormPage title="New report" submitLabel="Create report" cancelHref="/reports"
                 {errors} onsubmit={save}>
         <Field id="name" label="Name" bind:value={name} error={nameError} />
       </FormPage> -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Button from '../Button.svelte';
	import ErrorSummary from '../ErrorSummary.svelte';
	import PageHeader from './PageHeader.svelte';

	let {
		title,
		description,
		submitLabel = 'Save',
		cancelHref,
		errors = [],
		busy = false,
		onsubmit,
		children
	}: {
		title: string;
		description?: string;
		/** Name the action ("Create report"), not "Submit". */
		submitLabel?: string;
		cancelHref?: string;
		/** One plain sentence per problem, with the `id` of the Field it concerns so the summary
		 * links to it (WCAG 2.4.3, 3.3.1); empty when the form is valid. */
		errors?: Array<{ field?: string; message: string }>;
		/** True while saving: the button is disabled so a double click cannot submit twice. */
		busy?: boolean;
		onsubmit: () => void | Promise<void>;
		children: Snippet;
	} = $props();

	function handle(event: SubmitEvent) {
		event.preventDefault();
		if (!busy) void onsubmit();
	}
</script>

<div class="govuk-grid-row">
	<div class="govuk-grid-column-two-thirds">
		{#if errors.length > 0}<ErrorSummary {errors} />{/if}

		<PageHeader {title} {description} />

		<form onsubmit={handle} novalidate>
			{@render children()}
			<div class="govuk-button-group">
				<Button type="submit" disabled={busy}>{busy ? 'Saving…' : submitLabel}</Button>
				{#if cancelHref}<a class="govuk-link" href={cancelHref}>Cancel</a>{/if}
			</div>
		</form>
	</div>
</div>
