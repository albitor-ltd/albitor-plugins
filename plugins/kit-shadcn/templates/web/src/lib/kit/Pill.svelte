<!-- A starting point, then yours: a short status or count. The tone is never colour alone: each one
     draws a glyph beside the word, and the word is the label.

       <Pill tone="success">Live</Pill>   <Pill tone="warning">Needs review</Pill> -->
<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import Circle from '@lucide/svelte/icons/circle';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import CircleX from '@lucide/svelte/icons/circle-x';
	import Info from '@lucide/svelte/icons/info';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import Icon from './Icon.svelte';

	type Tone = 'neutral' | 'success' | 'warning' | 'danger' | 'info';

	let { tone = 'neutral', children }: { tone?: Tone; children: Snippet } = $props();

	const glyphs: Record<Tone, Component<{ size?: number | string; class?: string }>> = {
		neutral: Circle,
		success: CircleCheck,
		warning: TriangleAlert,
		danger: CircleX,
		info: Info
	};
</script>

<span class="pill {tone}" data-kit="pill" data-tone={tone}>
	<Icon icon={glyphs[tone]} size={14} />
	{@render children()}
</span>

<style>
	.pill {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		padding: 0.125rem var(--space-2);
		border-radius: calc(var(--radius) * 2);
		font-size: var(--text-small);
		font-weight: 500;
		white-space: nowrap;
	}
	.neutral {
		background: var(--muted);
		color: var(--foreground);
	}
	.success {
		background: var(--success);
		color: var(--on-success);
	}
	.warning {
		background: var(--warning);
		color: var(--on-warning);
	}
	.danger {
		background: var(--destructive);
		color: var(--destructive-foreground);
	}
	.info {
		background: var(--second);
		color: var(--on-second);
	}
</style>
