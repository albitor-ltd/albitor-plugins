<!-- Seeded by Albitor (kit-carbon), then yours: a labelled text control on Carbon's TextInput and
     TextArea. The label, the hint and the error are tied to the control, so a screen reader
     announces all three (Carbon alone drops the hint once there is an error). Never use a
     placeholder as the label. -->
<script lang="ts">
	import { TextArea, TextInput } from 'carbon-components-svelte';
	import type { HTMLInputAttributes } from 'svelte/elements';

	type Props = Omit<HTMLInputAttributes, 'value' | 'id'> & {
		label: string;
		/** Give it a name when an error summary links here (`{ field: 'name', message }`). */
		id?: string;
		value?: string;
		/** Help shown under the control, read before the error. */
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
	// Carbon derives the error id from the control's id. It hides its own helper text once there is
	// an error, so the hint is rendered here and stays.
	const hintId = $derived(`${uid}-hint`);
	const errorId = $derived(`error-${uid}`);
	const describedBy = $derived(
		[hint ? hintId : '', error ? errorId : '', extraDescribedBy ?? ''].filter(Boolean).join(' ') ||
			undefined
	);
</script>

{#if multiline}
	<TextArea
		{...rest as Record<string, unknown>}
		id={uid}
		{rows}
		labelText={label}
		invalid={Boolean(error)}
		invalidText={error ?? ''}
		aria-describedby={describedBy}
		bind:value
	/>
{:else}
	<TextInput
		{...rest as Record<string, unknown>}
		id={uid}
		type={type ?? 'text'}
		labelText={label}
		invalid={Boolean(error)}
		invalidText={error ?? ''}
		aria-describedby={describedBy}
		bind:value
	/>
{/if}

{#if hint}<p class="bx--form__helper-text hint" id={hintId}>{hint}</p>{/if}

<style>
	.hint {
		margin: -0.5rem 0 1rem;
	}
</style>
