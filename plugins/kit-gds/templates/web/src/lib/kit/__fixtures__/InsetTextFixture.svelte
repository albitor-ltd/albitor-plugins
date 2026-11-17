<!-- Test support: InsetText in the shell, for insettext.test.ts alone. Delete it with InsetText.svelte and insettext.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import { createRawSnippet } from 'svelte';
	import AppShell from '../AppShell.svelte';
	import InsetText from '../InsetText.svelte';
	import { fixtureNav } from './nav';

	const note = createRawSnippet(() => ({ render: () => '<span>You have until 31 March to reply.</span>' }));
</script>

<AppShell nav={fixtureNav()} path="/reports" title="Example app">
	<h1 class="govuk-heading-l">Reports</h1>
	<InsetText children={note} />
</AppShell>
