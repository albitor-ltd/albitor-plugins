<!-- Test support: TaskList in the shell, for tasklist.test.ts alone. Delete it with TaskList.svelte and tasklist.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import TaskList from '../TaskList.svelte';
	import { fixtureNav } from './nav';
</script>

<AppShell nav={fixtureNav()} path="/reports" title="Example app">
	<h1 class="govuk-heading-l">Apply for a licence</h1>
	<TaskList
		items={[
			{ title: 'Your details', href: '/details', hint: 'Name and email', status: 'completed' },
			{ title: 'Your premises', href: '/premises', status: 'in-progress' },
			{ title: 'Payment', href: '/payment', status: 'not-started' },
			{ title: 'Declaration', status: 'cannot-start' }
		]}
	/>
</AppShell>
