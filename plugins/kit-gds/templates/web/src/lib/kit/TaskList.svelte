<!-- A starting point, then yours: the GOV.UK task list (Frontend 6.4.0 markup) for a long
     transaction made of separate tasks. Each task has one status: "Completed" as plain text,
     "In progress" and "Not started" as a Tag, "Cannot start yet" as grey text with no link. The
     status is always words, never colour alone. The link's accessible description is its hint and
     its status, so a screen reader hears "Your details, Not started". Say how many tasks are done
     in a sentence above the list ("You have completed 1 of 3 sections"). -->
<script module lang="ts">
	export type TaskStatus = 'completed' | 'in-progress' | 'not-started' | 'cannot-start';

	export interface Task {
		title: string;
		/** Where the task starts. Leave it out for a task that cannot start yet. */
		href?: string;
		hint?: string;
		status: TaskStatus;
	}

	export const taskStatusLabels: Record<TaskStatus, string> = {
		completed: 'Completed',
		'in-progress': 'In progress',
		'not-started': 'Not started',
		'cannot-start': 'Cannot start yet'
	};
</script>

<script lang="ts">
	import Tag from './Tag.svelte';

	let { items }: { items: Task[] } = $props();

	const uid = $props.id();
</script>

<ul class="govuk-task-list">
	{#each items as item, i (i)}
		{@const hint = `${uid}-${i + 1}-hint`}
		{@const status = `${uid}-${i + 1}-status`}
		<li class={['govuk-task-list__item', item.href && 'govuk-task-list__item--with-link']}>
			<div class="govuk-task-list__name-and-hint">
				{#if item.href}
					<a
						class="govuk-link govuk-task-list__link"
						href={item.href}
						aria-describedby={`${item.hint ? hint + ' ' : ''}${status}`}>{item.title}</a
					>
				{:else}
					<div>{item.title}</div>
				{/if}
				{#if item.hint}<div id={hint} class="govuk-task-list__hint">{item.hint}</div>{/if}
			</div>
			<div
				class={['govuk-task-list__status', item.status === 'cannot-start' && 'govuk-task-list__status--cannot-start-yet']}
				id={status}
			>
				{#if item.status === 'in-progress' || item.status === 'not-started'}
					<Tag>{taskStatusLabels[item.status]}</Tag>
				{:else}
					{taskStatusLabels[item.status]}
				{/if}
			</div>
		</li>
	{/each}
</ul>
