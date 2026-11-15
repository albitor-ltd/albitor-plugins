<!-- Empty-state art for a first-run empty list: a starting point, then yours. Decorative only:
     the sentence and the action beside it do the work, so it carries no text and no meaning.
     Colour comes from the app's tokens, never a hex, so it follows light and dark. The three
     --art-* lines are the only place that names the kit's tokens: point them at the app's own
     (this uses the shadcn kit's --primary, --muted and --border). -->
<script lang="ts">
	let { size = 160 }: { size?: number } = $props();
</script>

<svg
	class="art"
	width={size}
	height={size}
	viewBox="0 0 160 160"
	aria-hidden="true"
	focusable="false"
>
	<!-- consumer: round, soft and overlapping; nothing sharp -->
	<circle class="tint" cx="80" cy="82" r="64" />
	<rect class="card" x="38" y="52" width="84" height="64" rx="24" />
	<circle class="mark" cx="80" cy="84" r="16" />
	<circle class="spark" cx="122" cy="38" r="9" />
	<circle class="spark" cx="34" cy="120" r="6" />
</svg>

<style>
	.art {
		--art-accent: var(--primary);
		--art-soft: var(--muted);
		--art-line: var(--border);
		display: block;
		margin-inline: auto;
	}
	.tint {
		fill: var(--art-accent);
		opacity: 0.12;
	}
	.card {
		fill: var(--art-soft);
		stroke: var(--art-line);
		stroke-width: 2;
	}
	.mark {
		fill: var(--art-accent);
		opacity: 0.85;
	}
	.spark {
		fill: var(--art-accent);
		opacity: 0.35;
	}
</style>
