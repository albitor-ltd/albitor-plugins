<!-- The loading and failed states of a page, shown where the content will appear. Loading is a
     polite status; a failure is a GOV.UK notification banner with the alert role and a retry,
     because a message that is gone once dismissed cannot be found again. -->
<script lang="ts">
	import Button from '../Button.svelte';

	let {
		status,
		loadingLabel = 'Loading',
		errorMessage = 'Something went wrong loading this page.',
		onretry
	}: {
		status: 'loading' | 'error';
		loadingLabel?: string;
		/** A sentence a person can act on. Never a raw server body. */
		errorMessage?: string;
		onretry?: () => void;
	} = $props();

	const uid = $props.id();
</script>

{#if status === 'loading'}
	<p role="status" class="govuk-body">{loadingLabel}…</p>
{:else}
	<div class="govuk-notification-banner" role="alert" aria-labelledby={`${uid}-title`}>
		<div class="govuk-notification-banner__header">
			<h2 class="govuk-notification-banner__title" id={`${uid}-title`}>There is a problem</h2>
		</div>
		<div class="govuk-notification-banner__content">
			<p class="govuk-notification-banner__heading">{errorMessage}</p>
			{#if onretry}<Button variant="secondary" onclick={onretry}>Try again</Button>{/if}
		</div>
	</div>
{/if}
