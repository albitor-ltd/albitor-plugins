<!-- The way back to a hub. In an app whose first screen is a hub of tiles (a portal) there is no
     bar of destinations, so every other page keeps this link at the top: nothing is hidden on a
     phone and a person is never stranded. A page that uses PageHeader can pass the same link as
     its `back`; use this on a page whose header is its own (a step page, a confirmation).

       <BackToHub />                                  to "/", labelled "Back to your account"
       <BackToHub href="/account" label="Back to your account" /> -->
<script lang="ts">
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Icon from '../Icon.svelte';

	let {
		href = '/',
		label = 'Back to your account'
	}: { href?: string; label?: string } = $props();
</script>

<p class="back"><a {href}><Icon icon={ArrowLeft} size={16} /> {label}</a></p>

<style>
	.back {
		margin: 0 0 var(--space-3);
	}
	a {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		min-height: 2.75rem;
		color: var(--link);
	}
	a:focus-visible {
		outline: var(--focus-width) solid var(--ring);
		outline-offset: 2px;
	}
</style>
