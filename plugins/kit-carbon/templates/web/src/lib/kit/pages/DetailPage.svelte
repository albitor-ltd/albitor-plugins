<!-- Seeded by Albitor (kit-carbon), then yours: one record. A header with a link back to the list
     and the record's actions, the facts as a description list (term and value stay paired for a
     screen reader), then any related sections as children. A destructive action opens a Dialog;
     never delete on one click.

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
		/** Related sections: give each its own `<h2>`. */
		children?: Snippet;
	} = $props();
</script>

<PageHeader {title} {description} {back} {actions} />

{#if status !== 'ready'}
	<StateNotice {status} {errorMessage} {onretry} loadingLabel="Loading details" />
{:else}
	<dl>
		{#each fields as field (field.label)}
			<div>
				<dt>{field.label}</dt>
				<dd>{field.value === '' ? 'Not set' : field.value}</dd>
			</div>
		{/each}
	</dl>
	{#if children}<div class="related">{@render children()}</div>{/if}
{/if}

<style>
	dl {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
		gap: 1rem 1.5rem;
		margin: 0 0 2rem;
	}
	dt {
		color: var(--cds-text-secondary);
		font-size: 0.875rem;
	}
	dd {
		margin: 0.125rem 0 0;
		font-weight: 600;
		overflow-wrap: anywhere;
	}
	.related :global(h2) {
		font-size: 1.25rem;
	}
</style>
