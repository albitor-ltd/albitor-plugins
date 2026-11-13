<!-- Test support: Field in the shell, for field.test.ts alone. Delete it with Field.svelte and
     field.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import Field from '../Field.svelte';
	import { fixtureNav } from './nav';

	let email = $state('');
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1 class="govuk-heading-l">Reports</h1>
	<form onsubmit={(e) => e.preventDefault()}>
		<Field
			id="email"
			label="Email address"
			type="email"
			bind:value={email}
			autocomplete="email"
			hint="We only use it to send the report."
			error="Enter an email address, like name@example.com"
		/>
		<Field label="Notes" multiline aria-describedby="notes-help" />
		<p class="govuk-body" id="notes-help">Notes are visible to your team.</p>
	</form>
</AppShell>
