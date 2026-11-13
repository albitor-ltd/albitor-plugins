<!-- Test support: ErrorSummary in the shell, for errorsummary.test.ts alone. Delete it with
     ErrorSummary.svelte and errorsummary.test.ts. The input is only the control the summary's entry
     links to. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import ErrorSummary from '../ErrorSummary.svelte';
	import { fixtureNav } from './nav';
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<ErrorSummary errors={[{ field: 'email', message: 'Enter an email address, like name@example.com' }]} />

	<h1 class="govuk-heading-l">Reports</h1>
	<form onsubmit={(e) => e.preventDefault()}>
		<label class="govuk-label" for="email">Email address</label>
		<input class="govuk-input" id="email" type="email" autocomplete="email" />
	</form>
</AppShell>
