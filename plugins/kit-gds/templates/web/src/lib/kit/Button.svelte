<!-- Seeded by Albitor (kit-gds), then yours: a GOV.UK button (govuk-button). A button is a <button>;
     to navigate, use a link. GOV.UK has one primary action per page: the green button. The variant
     names match the other kits so a page ports across: destructive is GOV.UK's red warning
     button, and ghost has no GOV.UK form, so it is the secondary (grey) button. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	type Props = HTMLButtonAttributes & {
		variant?: 'primary' | 'secondary' | 'destructive' | 'ghost';
		children: Snippet;
	};

	let { variant = 'primary', type = 'button', disabled, children, ...rest }: Props = $props();

	const modifier = $derived(
		variant === 'destructive'
			? ' govuk-button--warning'
			: variant === 'secondary' || variant === 'ghost'
				? ' govuk-button--secondary'
				: ''
	);
</script>

<button
	{type}
	class={'govuk-button' + modifier}
	{disabled}
	aria-disabled={disabled ? 'true' : undefined}
	data-module="govuk-button"
	{...rest}>{@render children()}</button
>
