<!-- Seeded by Albitor (kit-gds), then yours: a labelled text control in GOV.UK form-group markup. The
     label, the hint and the error are tied to the control, so a screen reader announces all
     three. Never use a placeholder as the label. -->
<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';

	type Props = Omit<HTMLInputAttributes, 'value' | 'id' | 'class'> & {
		label: string;
		/** Give it a name when an error summary links here (`{ field: 'name', message }`). */
		id?: string;
		value?: string;
		/** Help shown under the label, read before the error. */
		hint?: string;
		/** Set when the value is wrong. Say what to do, not just what is wrong. */
		error?: string;
		/** A textarea instead of an input. */
		multiline?: boolean;
		rows?: number;
	};

	let {
		id,
		label,
		value = $bindable(''),
		hint,
		error,
		multiline = false,
		rows = 5,
		type = 'text',
		'aria-describedby': extraDescribedBy,
		...rest
	}: Props = $props();

	const generated = $props.id();
	const uid = $derived(id ?? generated);
	const hintId = $derived(`${uid}-hint`);
	const errorId = $derived(`${uid}-error`);
	const describedBy = $derived(
		[hint ? hintId : '', error ? errorId : '', extraDescribedBy ?? ''].filter(Boolean).join(' ')
	);
</script>

<div class="govuk-form-group" class:govuk-form-group--error={error}>
	<label class="govuk-label" for={uid}>{label}</label>
	{#if hint}<div class="govuk-hint" id={hintId}>{hint}</div>{/if}
	{#if error}
		<p class="govuk-error-message" id={errorId}>
			<span class="govuk-visually-hidden">Error:</span>
			{error}
		</p>
	{/if}
	{#if multiline}
		<textarea
			{...rest as object}
			id={uid}
			{rows}
			class="govuk-textarea"
			class:govuk-textarea--error={error}
			bind:value
			aria-invalid={error ? 'true' : undefined}
			aria-describedby={describedBy || undefined}
		></textarea>
	{:else}
		<input
			{...rest}
			id={uid}
			{type}
			class="govuk-input"
			class:govuk-input--error={error}
			bind:value
			aria-invalid={error ? 'true' : undefined}
			aria-describedby={describedBy || undefined}
		/>
	{/if}
</div>
