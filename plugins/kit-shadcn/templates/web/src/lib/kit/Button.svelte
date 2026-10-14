<!-- Seeded by Albitor (kit-shadcn), then yours. A button is a <button>; to navigate, use a link. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	type Props = HTMLButtonAttributes & {
		variant?: 'primary' | 'secondary' | 'destructive' | 'ghost';
		children: Snippet;
	};

	let { variant = 'primary', type = 'button', children, ...rest }: Props = $props();
</script>

<button {type} class={variant} {...rest}>{@render children()}</button>

<style>
	button {
		min-height: 2.5rem;
		padding: 0 1rem;
		border: 1px solid transparent;
		border-radius: var(--radius);
		font: inherit;
		font-weight: 500;
		cursor: pointer;
	}
	.primary {
		background: var(--primary);
		color: var(--primary-foreground);
	}
	.secondary {
		background: var(--background);
		color: var(--foreground);
		border-color: var(--input);
	}
	.destructive {
		background: var(--destructive);
		color: var(--destructive-foreground);
	}
	.ghost {
		background: transparent;
		color: var(--foreground);
	}
	.ghost:hover,
	.secondary:hover {
		background: var(--muted);
	}
	button:focus-visible {
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}
	button:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}
</style>
