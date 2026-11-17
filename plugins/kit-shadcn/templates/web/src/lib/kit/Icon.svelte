<!-- A starting point, then yours: one way to draw a Lucide icon. Import the icon from its own
     path (a barrel import pulls in every icon) and pass the component:

       import House from '@lucide/svelte/icons/house';
       <Icon icon={House} />                       decorative: aria-hidden, the words are beside it
       <Icon icon={Bell} label="3 unread" />      meaningful on its own: role="img" with a name

     An icon that is the only content of a button is named on the button, not here. Icons take the
     text colour (currentColor). -->
<script lang="ts">
	import type { Component } from 'svelte';

	let {
		icon: Glyph,
		label,
		size = 20
	}: {
		icon: Component<{
			size?: number | string;
			class?: string;
			role?: 'img';
			'aria-label'?: string;
			'aria-hidden'?: 'true';
		}>;
		/** Set only when the icon carries meaning no nearby text does. */
		label?: string;
		size?: number;
	} = $props();
</script>

{#if label}
	<Glyph {size} class="kit-icon" role="img" aria-label={label} />
{:else}
	<Glyph {size} class="kit-icon" aria-hidden="true" />
{/if}

<style>
	:global(.kit-icon) {
		flex: none;
		vertical-align: middle;
	}
</style>
