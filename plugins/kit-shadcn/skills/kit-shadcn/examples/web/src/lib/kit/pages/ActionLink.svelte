<!-- A link that looks like a button, for navigation: a Button is a <button>, and a call to action
     that goes to another page is a link. Same tokens as Button, so the same contrast pairs apply.
     The target is at least 44px tall, whatever the look's control height.

       <ActionLink href="/start">Get started</ActionLink>
       <ActionLink href="/pricing" variant="secondary">See pricing</ActionLink> -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		href,
		variant = 'primary',
		children
	}: {
		href: string;
		variant?: 'primary' | 'secondary';
		children: Snippet;
	} = $props();
</script>

<a {href} class={variant}>{@render children()}</a>

<style>
	a {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
		min-height: max(var(--control-height), 2.75rem);
		padding: 0 var(--space-4);
		border: var(--border-width) solid transparent;
		border-radius: var(--radius);
		font-weight: 500;
		text-decoration: none;
		box-shadow: var(--shadow);
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
	.secondary:hover {
		background: var(--muted);
	}
	a:focus-visible {
		outline: var(--focus-width) solid var(--ring);
		outline-offset: 2px;
	}
</style>
