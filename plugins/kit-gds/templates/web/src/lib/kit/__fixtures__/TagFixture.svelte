<!-- Test support: Tag in the shell, for tag.test.ts alone. Delete it with Tag.svelte and tag.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import { createRawSnippet } from 'svelte';
	import AppShell from '../AppShell.svelte';
	import Tag from '../Tag.svelte';
	import { fixtureNav } from './nav';

	const label = createRawSnippet(() => ({ render: () => '<span>In progress</span>' }));
</script>

<AppShell nav={fixtureNav()} path="/reports" title="Example app">
	<h1 class="govuk-heading-l">Reports</h1>
	<p class="govuk-body">Status: <Tag children={label} /></p>
</AppShell>
