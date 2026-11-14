<!-- Empty-state art for a first-run empty list: a starting point, then yours. Decorative only:
     the sentence and the action beside it do the work, so it carries no text and no meaning.
     Colour comes from the app's tokens, never a hex, so it follows light and dark. The three
     --art-* lines are the only place that names the kit's tokens: point them at the app's own
     (this uses the shadcn kit's --primary, --muted and --border). -->
<script lang="ts">
	let { size = 152 }: { size?: number } = $props();
</script>

<svg
	class="art"
	width={size}
	height={size}
	viewBox="0 0 160 160"
	aria-hidden="true"
	focusable="false"
>
	<!-- startup: stacked layers with hairline edges and one accent mark; precise, not cute -->
	<rect class="layer back" x="42" y="30" width="84" height="60" rx="10" />
	<rect class="layer mid" x="34" y="50" width="84" height="60" rx="10" />
	<rect class="layer front" x="26" y="70" width="84" height="60" rx="10" />
	<rect class="mark" x="42" y="88" width="32" height="8" rx="4" />
	<rect class="line" x="42" y="104" width="52" height="6" rx="3" />
</svg>

<style>
	.art {
		--art-accent: var(--primary);
		--art-soft: var(--muted);
		--art-line: var(--border);
		display: block;
		margin-inline: auto;
	}
	.layer {
		fill: var(--art-soft);
		stroke: var(--art-line);
		stroke-width: 1;
	}
	.back {
		opacity: 0.5;
	}
	.mid {
		opacity: 0.75;
	}
	.mark {
		fill: var(--art-accent);
	}
	.line {
		fill: var(--art-line);
	}
</style>
