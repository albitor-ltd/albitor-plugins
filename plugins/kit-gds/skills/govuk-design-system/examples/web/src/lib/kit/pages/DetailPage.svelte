<!-- One record. A header with a link back to the list and the record's actions, the facts as a
     GOV.UK summary list (term and value stay paired for a screen reader), then any related
     sections as children. A destructive action opens a Dialog; never delete on one click.

       <DetailPage title={item.name} back={{ href: '/reports', label: 'All reports' }}
                   fields={[{ label: 'Owner', value: item.owner }]} {status} onretry={reload}>
         {#snippet actions()}<Button variant="destructive" onclick={askDelete}>Delete</Button>{/snippet}
       </DetailPage> -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import PageHeader from './PageHeader.svelte';
	import StateNotice from './StateNotice.svelte';

	let {
		title,
		description,
		back,
		fields,
		status = 'ready',
		errorMessage,
		onretry,
		actions,
		children
	}: {
		title: string;
		description?: string;
		back: { href: string; label: string };
		/** Values are text, never markup. */
		fields: Array<{ label: string; value: string | number }>;
		status?: 'loading' | 'error' | 'ready';
		errorMessage?: string;
		onretry?: () => void;
		actions?: Snippet;
		/** Related sections: give each its own `<h2 class="govuk-heading-m">`. */
		children?: Snippet;
	} = $props();
</script>

<PageHeader {title} {description} {back} {actions} />

{#if status !== 'ready'}
	<StateNotice {status} {errorMessage} {onretry} loadingLabel="Loading details" />
{:else}
	<dl class="govuk-summary-list">
		{#each fields as field (field.label)}
			<div class="govuk-summary-list__row">
				<dt class="govuk-summary-list__key">{field.label}</dt>
				<dd class="govuk-summary-list__value">{field.value === '' ? 'Not set' : field.value}</dd>
			</div>
		{/each}
	</dl>
	{#if children}<div>{@render children()}</div>{/if}
{/if}
