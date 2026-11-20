<!-- One item with a next step: the facts about it, then an enquiry or booking form beside them (below
     them on a phone). A header with a link back to the catalogue, a status Pill, the facts as a
     description list, an `about` section for longer text, and the form Card. On a failed submit the
     summary takes focus; on success the form is replaced by a confirmation that says what happens
     next, announced politely. The form's fields are the caller's: name, email, a date or a
     message, each a labelled Field.

       <ItemPage title={course.name} back={{ href: '/courses', label: 'All courses' }}
                 fields={[{ label: 'Where', value: 'Hall 2' }]} {errors} {sent} onsubmit={send}
                 enquiryTitle="Book a place" submitLabel="Book" sentMessage="We have emailed your booking.">
         {#snippet enquiry()}<Field id="email" label="Email" type="email" bind:value={email} />{/snippet}
       </ItemPage>

     Validate on the server too: the browser is not a boundary. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Button from '../Button.svelte';
	import Card from '../Card.svelte';
	import ErrorSummary from './ErrorSummary.svelte';
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
		badge,
		about,
		enquiryTitle = 'Enquire about this',
		enquiry,
		submitLabel = 'Send enquiry',
		errors = [],
		busy = false,
		sent = false,
		sentTitle = 'Thank you',
		sentMessage,
		onsubmit
	}: {
		title: string;
		description?: string;
		back: { href: string; label: string };
		/** Values are text, never markup. */
		fields: Array<{ label: string; value: string | number }>;
		status?: 'loading' | 'error' | 'ready';
		errorMessage?: string;
		onretry?: () => void;
		/** A Pill beside the title: availability, status. */
		badge?: Snippet;
		/** Longer text about the item; give it no <h1>. */
		about?: Snippet;
		enquiryTitle?: string;
		/** The form's labelled Fields. */
		enquiry: Snippet;
		/** Name the action ("Book a place"), not "Submit". */
		submitLabel?: string;
		/** One plain sentence per problem, with the `id` of the Field it concerns. */
		errors?: Array<{ field?: string; message: string }>;
		busy?: boolean;
		/** The enquiry went through: show the confirmation in place of the form. */
		sent?: boolean;
		sentTitle?: string;
		/** What happens next, in a sentence. */
		sentMessage: string;
		onsubmit: () => void | Promise<void>;
	} = $props();

	function handle(event: SubmitEvent) {
		event.preventDefault();
		if (!busy) void onsubmit();
	}
</script>

<PageHeader {title} {description} {back} {badge} />

{#if status !== 'ready'}
	<StateNotice {status} {errorMessage} {onretry} loadingLabel="Loading details" />
{:else}
	<div class="layout">
		<div class="facts">
			<Card>
				<dl>
					{#each fields as field (field.label)}
						<div>
							<dt>{field.label}</dt>
							<dd>{field.value === '' ? 'Not set' : field.value}</dd>
						</div>
					{/each}
				</dl>
			</Card>
			{#if about}<div class="about">{@render about()}</div>{/if}
		</div>
		<div class="action">
			{#if sent}
				<Card tone="second" title={sentTitle}>
					<p role="status" class="sent">{sentMessage}</p>
				</Card>
			{:else}
				<ErrorSummary {errors} />
				<Card title={enquiryTitle}>
					<form onsubmit={handle} novalidate>
						<div class="fields">{@render enquiry()}</div>
						<Button type="submit" disabled={busy}>{busy ? 'Sending…' : submitLabel}</Button>
					</form>
				</Card>
			{/if}
		</div>
	</div>
{/if}

<style>
	.layout {
		display: grid;
		grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr);
		gap: var(--space-6);
		align-items: start;
	}
	@media (width < 48rem) {
		.layout {
			grid-template-columns: 1fr;
		}
	}
	dl {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
		gap: var(--space-4) var(--space-6);
		margin: 0;
	}
	dt {
		color: var(--muted-foreground);
		font-size: var(--text-small);
	}
	dd {
		margin: 0.125rem 0 0;
		font-weight: 600;
		overflow-wrap: anywhere;
	}
	.about {
		margin-top: var(--space-6);
	}
	.about :global(h2) {
		font-size: var(--text-heading);
	}
	.fields {
		display: grid;
		gap: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.sent {
		margin: 0;
	}
</style>
