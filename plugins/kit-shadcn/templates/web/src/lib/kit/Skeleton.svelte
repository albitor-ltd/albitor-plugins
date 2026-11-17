<!-- A starting point, then yours: a placeholder shaped like what is loading. It is decorative
     and hidden from assistive technology; the region that holds skeletons says it is loading
     with `aria-busy="true"` and a visible or screen-reader-only "Loading" message.

       <div aria-busy="true"><Skeleton height="1.25rem" width="60%" /></div>

     The shimmer follows the look's motion (none when the look sets none, and none under reduced
     motion). -->
<script lang="ts">
	let {
		width = '100%',
		height = '1rem',
		circle = false
	}: { width?: string; height?: string; circle?: boolean } = $props();
</script>

<span
	class="skeleton"
	class:circle
	data-kit="skeleton"
	aria-hidden="true"
	style:inline-size={width}
	style:block-size={height}
></span>

<style>
	.skeleton {
		display: block;
		border-radius: var(--radius);
		background: linear-gradient(90deg, var(--muted) 25%, var(--border) 50%, var(--muted) 75%);
		background-size: 200% 100%;
		animation: shimmer 1s var(--motion-ease) infinite;
		animation-duration: calc(var(--motion-duration) * 8);
	}
	.circle {
		border-radius: 50%;
	}
	@keyframes shimmer {
		from {
			background-position: 200% 0;
		}
		to {
			background-position: -200% 0;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.skeleton {
			animation: none;
		}
	}
</style>
