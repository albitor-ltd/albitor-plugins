<!-- Seeded by Albitor (kit-carbon), then yours. A button is a <button>; to navigate, use a link.
     Wraps Carbon's Button, so the variants map onto Carbon's kinds. -->
<script lang="ts">
	import { Button } from 'carbon-components-svelte';
	import type { Component, Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	type Variant = 'primary' | 'secondary' | 'destructive' | 'ghost';

	type Props = HTMLButtonAttributes & {
		variant?: Variant;
		/** A Carbon icon (carbon-icons-svelte) shown after the text. */
		icon?: Component;
		children: Snippet;
	};

	const kinds = {
		primary: 'primary',
		secondary: 'secondary',
		destructive: 'danger',
		ghost: 'ghost'
	} as const;

	let { variant = 'primary', type = 'button', children, ...rest }: Props = $props();
</script>

<Button kind={kinds[variant]} type={type ?? 'button'} {...rest as Record<string, unknown>}>{@render children()}</Button>
