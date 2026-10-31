<!-- Starting point (kit-gds): test fixture for rootclasses.test.ts. A child that records the
     body classes it sees in its own onMount, as a component calling initAll() would. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import AppShell from './AppShell.svelte';

	onMount(() => {
		document.body.dataset.seen = document.body.className;
	});
</script>

<AppShell nav={[]} path="/" title="Probe">
	<p>probe</p>
</AppShell>
