<!-- Seeded by Albitor (kit-gds), then yours: the GOV.UK error summary (Frontend 6.4.0 markup). Show
     it at the top of the main content, above the h1, when a form fails validation, and give each
     failing Field the id its entry names. It takes focus when it appears, so a screen reader
     announces it; render it only while there are errors ({#if errors.length}), so it mounts anew
     each time a submit fails. Each message is the same text as the Field's own error. -->
<script lang="ts">
	import { onMount } from 'svelte';

	let {
		errors,
		title = 'There is a problem',
		description
	}: {
		/** `field` is the id of the control that is wrong; the entry links to it. */
		errors: Array<{ field?: string; message: string }>;
		title?: string;
		description?: string;
	} = $props();

	let el: HTMLDivElement | null = $state(null);

	onMount(() => el?.focus());
</script>

<div class="govuk-error-summary" tabindex="-1" bind:this={el}>
	<!-- The alert lives in a child so the focus move and the announcement do not race. -->
	<div role="alert">
		<h2 class="govuk-error-summary__title">{title}</h2>
		<div class="govuk-error-summary__body">
			{#if description}<p>{description}</p>{/if}
			<ul class="govuk-list govuk-error-summary__list">
				{#each errors as item, i (i)}
					<li>
						{#if item.field}<a href={`#${item.field}`}>{item.message}</a>{:else}{item.message}{/if}
					</li>
				{/each}
			</ul>
		</div>
	</div>
</div>
