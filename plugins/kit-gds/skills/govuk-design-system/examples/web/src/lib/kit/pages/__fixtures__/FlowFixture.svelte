<!-- Test support: the four pages of the task-list flow, one at a time, for flow.test.ts alone. Delete
     it with TaskListPage.svelte, QuestionPage.svelte, CheckAnswersPage.svelte,
     ConfirmationPage.svelte and flow.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import CheckAnswersPage from '../CheckAnswersPage.svelte';
	import ConfirmationPage from '../ConfirmationPage.svelte';
	import QuestionPage from '../QuestionPage.svelte';
	import TaskListPage from '../TaskListPage.svelte';
	import PagesShowcase from './PagesShowcase.svelte';

	let {
		step,
		error,
		allDone = false
	}: {
		step: 'tasks' | 'question' | 'check' | 'confirmation';
		error?: string;
		allDone?: boolean;
	} = $props();

	let name = $state('Sarah Philips');
	const noop = () => {};
</script>

<PagesShowcase>
	{#if step === 'tasks'}
		<TaskListPage
			title="Apply for a licence"
			checkHref="/apply/check"
			items={[
				{ title: 'Your name', href: '/apply/name', status: allDone ? 'completed' : 'not-started' },
				{ title: 'Your premises', href: '/apply/premises', status: allDone ? 'completed' : 'in-progress' },
				{ title: 'Declaration', status: allDone ? 'completed' : 'cannot-start' }
			]}
		/>
	{:else if step === 'question'}
		<QuestionPage
			question="What is your full name?"
			id="name"
			bind:value={name}
			hint="As it appears on your passport."
			{error}
			backHref="/apply"
			onsubmit={noop}
		/>
	{:else if step === 'check'}
		<CheckAnswersPage
			backHref="/apply"
			onsubmit={noop}
			rows={[
				{ key: 'Name', value: 'Sarah Philips', change: { href: '/apply/name', hidden: 'name' } },
				{ key: 'Premises', value: '1 High Street', change: { href: '/apply/premises', hidden: 'premises' } }
			]}
		/>
	{:else}
		<ConfirmationPage title="Application sent" reference="HDJ2123F">
			<p class="govuk-body">We have sent you a confirmation email.</p>
		</ConfirmationPage>
	{/if}
</PagesShowcase>
