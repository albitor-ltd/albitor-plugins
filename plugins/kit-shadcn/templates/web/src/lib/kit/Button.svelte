<!-- A starting point, then yours. A button is a <button>; to navigate, use a link. -->
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
	/* The hit area reaches 44px where the look draws a smaller control (28px, 36px). */
	button::after {
		content: '';
		position: absolute;
		inset-block: var(--hit-inset);
		inset-inline: 0;
	}
	button {
		min-height: var(--control-height);
		padding: 0 var(--space-4);
		border: var(--border-width) solid transparent;
		border-radius: var(--radius);
		font: inherit;
		font-weight: 500;
		cursor: pointer;
		position: relative;
		transition:
			background-color var(--motion-duration) var(--motion-ease),
			color var(--motion-duration) var(--motion-ease);
	}
	/* Each variant owns its elevation: the filled and outlined ones take the look's shadow, a
	   ghost has none, so it never floats over the surface it sits on. */
	.primary {
		background: var(--primary);
		color: var(--primary-foreground);
		box-shadow: var(--shadow);
	}
	.secondary {
		background: var(--background);
		color: var(--foreground);
		border-color: var(--input);
		box-shadow: var(--shadow);
	}
	.destructive {
		background: var(--destructive);
		color: var(--destructive-foreground);
		box-shadow: var(--shadow);
	}
	.ghost {
		background: transparent;
		color: var(--foreground);
		box-shadow: none;
	}
	.ghost:hover,
	.secondary:hover {
		background: var(--muted);
	}
	button:focus-visible {
		outline: var(--focus-width) solid var(--ring);
		outline-offset: 2px;
	}
	button:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}
</style>
