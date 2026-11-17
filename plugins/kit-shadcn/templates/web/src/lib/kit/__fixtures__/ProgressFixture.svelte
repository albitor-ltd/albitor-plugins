<!-- Test support: Progress in the shell, for progress.test.ts alone. Delete it with Progress.svelte
     and progress.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import Progress from '../Progress.svelte';
	import { fixtureNav } from './nav';
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1>Setup</h1>
	<Progress label="Storage used" value={62} showValue />
	<Progress label="Setup steps" value={3} max={5} variant="ring" showValue />
	<Progress label="Importing" value={null} />
</AppShell>
