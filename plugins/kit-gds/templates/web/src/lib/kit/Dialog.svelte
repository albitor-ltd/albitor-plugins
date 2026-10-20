<!-- Seeded by Albitor (kit-gds), then yours: a modal dialog. GOV.UK has no modal component, so this is
     a GOV.UK-styled native <dialog>, which the browser gives the focus trap, Escape, the inert page
     behind it and returning focus to the control that opened it. Always give it a title and a
     description: the title is the dialog's accessible name and the description is read with it.
     Prefer a page or an inline pattern to a modal wherever the journey allows. -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		open = $bindable(false),
		title,
		description,
		children,
		actions
	}: {
		open?: boolean;
		title: string;
		description: string;
		children?: Snippet;
		actions?: Snippet;
	} = $props();

	const uid = $props.id();
	let el: HTMLDialogElement | null = $state(null);

	$effect(() => {
		if (!el) return;
		if (open && !el.open) {
			// jsdom has no showModal; a real browser always does.
			if (typeof el.showModal === 'function') el.showModal();
			else el.setAttribute('open', '');
		} else if (!open && el.open) {
			if (typeof el.close === 'function') el.close();
			else el.removeAttribute('open');
		}
	});
</script>

<dialog
	bind:this={el}
	class="kit-dialog"
	aria-labelledby={`${uid}-title`}
	aria-describedby={`${uid}-description`}
	onclose={() => (open = false)}
>
	<h2 class="govuk-heading-m" id={`${uid}-title`}>{title}</h2>
	<p class="govuk-body" id={`${uid}-description`}>{description}</p>
	{#if children}{@render children()}{/if}
	<div class="govuk-button-group">
		{#if actions}{@render actions()}{/if}
		<button type="button" class="govuk-button govuk-button--secondary" onclick={() => (open = false)}>
			Close
		</button>
	</div>
</dialog>

<style>
	.kit-dialog {
		box-sizing: border-box;
		width: min(40rem, calc(100vw - 2rem));
		max-height: calc(100vh - 2rem);
		overflow: auto;
		padding: 30px;
		border: 5px solid #0b0c0c;
		border-radius: 0;
		background: #fff;
		color: #0b0c0c;
	}
	.kit-dialog::backdrop {
		background: rgb(11 12 12 / 0.7);
	}
</style>
