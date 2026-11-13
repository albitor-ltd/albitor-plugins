<!-- Test support: Button in every variant, in the shell, for button.test.ts alone. Delete it with
     Button.svelte and button.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import Send from 'carbon-icons-svelte/lib/Send.svelte';
	import AppShell from '../AppShell.svelte';
	import Button from '../Button.svelte';
	import { fixtureNav } from './nav';
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1>Reports</h1>
	<form onsubmit={(e) => e.preventDefault()}>
		<Button type="submit" icon={Send}>Send report</Button>
		<Button variant="secondary">Save draft</Button>
		<Button variant="ghost">Reset</Button>
		<Button variant="destructive">Delete report</Button>
	</form>
</AppShell>
