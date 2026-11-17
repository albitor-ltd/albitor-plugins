<!-- Step 4: confirmation. Say it is done, give the reference, then what happens next, and never make
     the page the only record: send it by email too. No Back link: the transaction is over. The
     GOV.UK panel is a green block, and this kit is plain, so the heading and a bold reference do the
     same job.

       <ConfirmationPage title="Application sent" reference="HDJ2123F">
         <p class="govuk-body">We have sent you a confirmation email.</p>
       </ConfirmationPage> -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		reference,
		children
	}: {
		title: string;
		/** The person's reference number, if there is one. */
		reference?: string;
		/** What happens next, as body text. */
		children: Snippet;
	} = $props();
</script>

<div class="govuk-grid-row">
	<div class="govuk-grid-column-two-thirds">
		<h1 class="govuk-heading-l">{title}</h1>
		{#if reference}
			<p class="govuk-body">Your reference number<br /><strong>{reference}</strong></p>
		{/if}
		<h2 class="govuk-heading-m">What happens next</h2>
		{@render children()}
	</div>
</div>
