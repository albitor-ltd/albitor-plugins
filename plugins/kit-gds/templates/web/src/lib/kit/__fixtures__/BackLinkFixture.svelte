<!-- Test support: BackLink in the shell, for backlink.test.ts alone. Delete it with BackLink.svelte and backlink.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import BackLink from '../BackLink.svelte';
	import { fixtureNav } from './nav';
</script>

<AppShell nav={fixtureNav()} path="/reports" title="Example app">
	<BackLink href="/" />
	<h1 class="govuk-heading-l">Reports</h1>
</AppShell>
