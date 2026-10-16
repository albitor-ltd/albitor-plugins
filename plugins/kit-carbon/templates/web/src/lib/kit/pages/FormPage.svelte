<!-- Seeded by Albitor (kit-carbon), then yours: create or edit one record. Put labelled Fields in the
     children. On a failed submit, pass `errors`: the summary appears above the form and takes focus
     so a screen reader announces what to fix (WCAG 3.3.1), and each Field also gets its own `error`
     next to the control. Validate on the server too: the browser is not a boundary.

       <FormPage title="New report" submitLabel="Create report" cancelHref="/reports"
                 {errors} onsubmit={save}>
         <Field label="Name" bind:value={name} error={nameError} />
       </FormPage> -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Button from '../Button.svelte';
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
		/** One plain sentence per problem; empty when the form is valid. */
		errors?: string[];
		/** True while saving: the button is disabled so a double click cannot submit twice. */
		busy?: boolean;
		onsubmit: () => void | Promise<void>;
		children: Snippet;
	} = $props();

	let summary: HTMLElement | undefined = $state();

	$effect(() => {
		if (errors.length > 0) summary?.focus();
	});

	function handle(event: SubmitEvent) {
		event.preventDefault();
		if (!busy) void onsubmit();
	}
</script>

<PageHeader {title} {description} />

{#if errors.length > 0}
	<div class="summary" role="alert" tabindex="-1" bind:this={summary}>
		<h2>There is a problem</h2>
		<ul>
			{#each errors as message (message)}<li>{message}</li>{/each}
		</ul>
	</div>
{/if}

<form onsubmit={handle} novalidate>
	{@render children()}
	<div class="buttons">
		<Button type="submit" disabled={busy}>{busy ? 'Saving…' : submitLabel}</Button>
		{#if cancelHref}<a href={cancelHref}>Cancel</a>{/if}
	</div>
</form>

<style>
	form {
		max-width: 40rem;
	}
	.summary {
		max-width: 40rem;
		margin-bottom: 1.5rem;
		padding: 1rem;
		border: 2px solid var(--cds-text-error);
		border-radius: 0;
	}
	.summary:focus-visible {
		outline: 2px solid var(--cds-focus);
		outline-offset: 2px;
	}
	.summary h2 {
		margin: 0 0 0.5rem;
		font-size: 1.25rem;
	}
	.summary ul {
		margin: 0;
		padding-inline-start: 1.25rem;
	}
	.buttons {
		display: flex;
		align-items: center;
		gap: 1rem;
	}
	.buttons a {
		color: var(--cds-link-primary);
	}
</style>
