<!-- Empty-state art for a first-run screen: a starting point, then yours. Decorative only: the
     sentence and the action beside it do the work, so it carries no text and no meaning. Colour
     comes from the app's tokens, never a hex, so it follows light and dark and the app's own
     colours. The --art-* lines are the only place that names the tokens (the shadcn kit's
     --primary, --second, --muted and --border): point them at the app's own if they differ. -->
<script lang="ts">
	let { size = 144 }: { size?: number } = $props();
</script>

<svg
	class="art"
	width={size}
	height={size}
	viewBox="0 0 160 160"
	aria-hidden="true"
	focusable="false"
>
	<!-- business: a ruled page with square corners and straight lines; calm and quiet -->
	<rect class="page" x="36" y="24" width="88" height="112" rx="2" />
	<rect class="bar" x="50" y="42" width="40" height="8" rx="2" />
	<line class="rule" x1="50" y1="70" x2="110" y2="70" />
	<line class="rule" x1="50" y1="88" x2="110" y2="88" />
	<line class="rule" x1="50" y1="106" x2="86" y2="106" />
	<rect class="flag" x="98" y="100" width="12" height="12" rx="2" />
</svg>

<style>
	.art {
		--art-accent: var(--primary);
		--art-second: var(--second);
		--art-soft: var(--muted);
		--art-line: var(--border);
		display: block;
		margin-inline: auto;
	}
	.page {
		fill: var(--art-soft);
		stroke: var(--art-line);
		stroke-width: 2;
	}
	.bar {
		fill: var(--art-accent);
		opacity: 0.8;
	}
	.rule {
		stroke: var(--art-line);
		stroke-width: 2;
		stroke-linecap: square;
	}
	.flag {
		fill: var(--art-second);
	}
</style>
