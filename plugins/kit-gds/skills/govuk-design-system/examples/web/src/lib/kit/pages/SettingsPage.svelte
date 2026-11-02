<!-- Settings in titled groups. Each group is a SettingsSection (a named region holding the fields
     that belong together), and the page has one Save: settings are saved together, not field by
     field. A group of related choices (radios, checkboxes) goes in a `<fieldset>` with a
     `<legend>`, as GOV.UK's checkboxes and radios do. Pass `errors` as `{ field, message }` (field
     is a Field's `id`) so the error summary links to the control.

       <SettingsPage title="Settings" {errors} onsubmit={save}>
         {#snippet sections()}
           <SettingsSection title="Profile"><Field id="display-name" label="Display name" bind:value={name} /></SettingsSection>
         {/snippet}
       </SettingsPage> -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Button from '../Button.svelte';
	import ErrorSummary from '../ErrorSummary.svelte';
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
			{@render sections()}
			<Button type="submit" disabled={busy}>{busy ? 'Saving…' : submitLabel}</Button>
		</form>
	</div>
</div>
