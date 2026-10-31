<!-- Starting point (web-svelte): tells the person when the browser loses its network. -->
<script lang="ts">
	let online = $state(true);

	$effect(() => {
		online = navigator.onLine;
		const up = () => (online = true);
		const down = () => (online = false);
		window.addEventListener('online', up);
		window.addEventListener('offline', down);
		return () => {
			window.removeEventListener('online', up);
			window.removeEventListener('offline', down);
		};
	});
</script>

<div role="status" class="shell-offline" hidden={online}>
	{#if !online}You are offline. Changes will not save until you reconnect.{/if}
</div>

<style>
	.shell-offline {
		padding: 0.5rem 1rem;
		background: #fff4cc;
		color: #1a1a1a;
		border-bottom: 1px solid #c9a227;
	}
	.shell-offline[hidden] {
		display: none;
	}
</style>
