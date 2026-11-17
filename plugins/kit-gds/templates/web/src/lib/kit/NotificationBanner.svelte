<!-- A starting point, then yours: the GOV.UK notification banner (Frontend 6.4.0 markup). Use it at
     the top of the main content, above the h1, to tell the person something about the service or
     the thing they just did. 'success' (green) is announced with the alert role and takes focus
     when it appears, so render it only once there is something to say ({#if done}); 'important'
     (blue) is a region landmark and takes no focus. A single line goes in `heading`
     (rendered with `govuk-notification-banner__heading`, per GOV.UK Frontend); give a longer body
     as children. Never use it for form errors: that is the error summary. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import type { Snippet } from 'svelte';

	let {
		type = 'important',
		title,
		headingLevel = 2,
		heading,
		children
	}: {
		type?: 'important' | 'success';
		/** The banner's title; "Important" or "Success" when left out. */
		title?: string;
		headingLevel?: 2 | 3 | 4;
		/** Single-line content, set in the banner's heading style. Use children for anything longer. */
		heading?: string;
		children?: Snippet;
	} = $props();

	const uid = $props.id();
	const success = $derived(type === 'success');
	const heading = $derived(`h${headingLevel}`);
	let el: HTMLDivElement | null = $state(null);

	onMount(() => {
		if (success) el?.focus();
	});
</script>

<!-- Only the success banner takes focus, and only from script (as Frontend does). -->
<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div
	class="govuk-notification-banner"
	class:govuk-notification-banner--success={success}
	role={success ? 'alert' : 'region'}
	aria-labelledby={`${uid}-title`}
	tabindex={success ? -1 : undefined}
	bind:this={el}
>
	<div class="govuk-notification-banner__header">
		<svelte:element this={heading} class="govuk-notification-banner__title" id={`${uid}-title`}>
			{title ?? (success ? 'Success' : 'Important')}
		</svelte:element>
	</div>
	<div class="govuk-notification-banner__content">
		{#if heading}<p class="govuk-notification-banner__heading">{heading}</p>{/if}
		{@render children?.()}
	</div>
</div>
