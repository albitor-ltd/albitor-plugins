<!-- Seeded by Albitor (kit-carbon), then yours: settings in titled groups. Each group is a
     SettingsSection (a named region holding the fields that belong together), and the page has one
     Save: settings are saved together, not field by field. A group of related choices (radios,
     checkboxes) goes in a `<fieldset>` with a `<legend>`. Confirm a save with a toast. Pass `errors` as `{ field, message }` (field is a Field's `id`)
     so the summary links to the control.

       <SettingsPage title="Settings" {errors} onsubmit={save}>
         {#snippet sections()}
           <SettingsSection title="Profile"><Field id="display-name" label="Display name" bind:value={name} /></SettingsSection>
         {/snippet}
       </SettingsPage> -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Button from '../Button.svelte';
	import PageHeader from './PageHeader.svelte';

	let {
		title,
		description,
		submitLabel = 'Save changes',
		errors = [],
		busy = false,
		onsubmit,
		sections
	}: {
		title: string;
		description?: string;
		submitLabel?: string;
		/** One plain sentence per problem; empty when the settings are valid. */
		errors?: Array<{ field?: string; message: string }>;
		busy?: boolean;
		onsubmit: () => void | Promise<void>;
		/** One `<SettingsSection>` per group. */
		sections: Snippet;
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
			{#each errors as error, i (i)}
				<li>{#if error.field}<a href="#{error.field}">{error.message}</a>{:else}{error.message}{/if}</li>
			{/each}
		</ul>
	</div>
{/if}

<form onsubmit={handle} novalidate>
	{@render sections()}
	<Button type="submit" disabled={busy}>{busy ? 'Saving…' : submitLabel}</Button>
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
</style>
