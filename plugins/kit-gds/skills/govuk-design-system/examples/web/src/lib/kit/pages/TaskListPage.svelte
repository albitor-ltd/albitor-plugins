<!-- Step 1 of a long transaction: the task list, where the person starts and returns. Each task is
     a link to its first question page; the status is words (Completed, In progress, Not started,
     Cannot start yet), never colour alone. It says how many are done, and offers the check-answers
     page only once every task is complete.

       <TaskListPage title="Apply for a licence" items={tasks} checkHref="/apply/check" /> -->
<script lang="ts">
	import TaskList from '../TaskList.svelte';
	import type { Task } from '../TaskList.svelte';
	import PageHeader from './PageHeader.svelte';

	let {
		title,
		description,
		items,
		checkHref
	}: {
		title: string;
		description?: string;
		items: Task[];
		/** The check-answers page. Offered only when every task is completed. */
		checkHref: string;
	} = $props();

	const done = $derived(items.filter((t) => t.status === 'completed').length);
</script>

<div class="govuk-grid-row">
	<div class="govuk-grid-column-two-thirds">
		<PageHeader {title} {description} />
		<p class="govuk-body">You have completed {done} of {items.length} sections.</p>
		<TaskList {items} />
		{#if done === items.length}
			<a class="govuk-button" href={checkHref} role="button" draggable="false" data-module="govuk-button">
				Check your answers
			</a>
		{/if}
	</div>
</div>
