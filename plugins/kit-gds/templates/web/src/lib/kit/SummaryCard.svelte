<!-- A starting point, then yours: the GOV.UK summary card (Frontend 6.4.0 markup): a titled summary
     list for one of several things of the same kind (each applicant, each address). Its title is a
     heading, so pick `headingLevel` to follow the page's own; each card action names the card in
     hidden text. -->
<script lang="ts">
	import SummaryList from './SummaryList.svelte';
	import type { SummaryRow } from './SummaryList.svelte';

	let {
		title,
		headingLevel = 2,
		actions = [],
		rows
	}: {
		title: string;
		headingLevel?: 2 | 3 | 4;
		/** Links for the whole card, such as Delete or Change. */
		actions?: Array<{ href: string; text: string }>;
		rows: SummaryRow[];
	} = $props();

	const heading = $derived(`h${headingLevel}`);
</script>

<div class="govuk-summary-card">
	<div class="govuk-summary-card__title-wrapper">
		<svelte:element this={heading} class="govuk-summary-card__title">{title}</svelte:element>
		{#if actions.length === 1}
			<div class="govuk-summary-card__actions">
				<a class="govuk-link" href={actions[0].href}>
					{actions[0].text}<span class="govuk-visually-hidden">{` (${title})`}</span>
				</a>
			</div>
		{:else if actions.length > 1}
			<ul class="govuk-summary-card__actions">
				{#each actions as action (action.href)}
					<li class="govuk-summary-card__action">
						<a class="govuk-link" href={action.href}>
							{action.text}<span class="govuk-visually-hidden">{` (${title})`}</span>
						</a>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
	<div class="govuk-summary-card__content">
		<SummaryList {rows} />
	</div>
</div>
