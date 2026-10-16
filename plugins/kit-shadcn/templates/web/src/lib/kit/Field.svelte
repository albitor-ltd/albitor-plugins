<!-- Seeded by Albitor (kit-shadcn), then yours: a labelled text control. The label, the hint and the
     error are tied to the control, so a screen reader announces all three. Never use a
     placeholder as the label. -->
<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';

	type Props = Omit<HTMLInputAttributes, 'value' | 'id'> & {
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
		rows = 4,
		type = 'text',
		'aria-describedby': extraDescribedBy,
		...rest
	}: Props = $props();

	const generated = $props.id();
	const uid = $derived(id ?? generated);
	const hintId = $derived(`${uid}-hint`);
	const errorId = $derived(`${uid}-error`);
	const describedBy = $derived([hint ? hintId : '', error ? errorId : '', extraDescribedBy ?? '']
			.filter(Boolean)
			.join(' '));
</script>

<div class="field">
	<label for={uid}>{label}</label>
	{#if hint}<p class="hint" id={hintId}>{hint}</p>{/if}
	{#if error}<p class="error" id={errorId}><span class="sr">Error: </span>{error}</p>{/if}
	{#if multiline}
		<textarea
			{...rest as object}
			id={uid}
			{rows}
			bind:value
			aria-invalid={error ? 'true' : undefined}
			aria-describedby={describedBy || undefined}
		></textarea>
	{:else}
		<input
			{...rest}
			id={uid}
			{type}
			bind:value
			aria-invalid={error ? 'true' : undefined}
			aria-describedby={describedBy || undefined}
		/>
	{/if}
</div>

<style>
	.field {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		margin-bottom: 1rem;
	}
	label {
		font-weight: 600;
	}
	.hint {
		margin: 0;
		color: var(--muted-foreground);
		font-size: var(--text-small);
	}
	.error {
		margin: 0;
		color: var(--destructive);
		font-size: var(--text-small);
		font-weight: 600;
	}
	.sr {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}
	input,
	textarea {
		min-height: 2.5rem;
		padding: 0.5rem 0.75rem;
		border: 1px solid var(--input);
		border-radius: var(--radius);
		background: var(--background);
		color: inherit;
		font: inherit;
	}
	input[aria-invalid='true'],
	textarea[aria-invalid='true'] {
		border-color: var(--destructive);
		border-width: 2px;
	}
	input:focus-visible,
	textarea:focus-visible {
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}
</style>
