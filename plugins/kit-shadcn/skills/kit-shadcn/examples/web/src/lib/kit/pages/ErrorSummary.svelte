<!-- The error summary of a form page: on a failed submit it appears above the form and takes focus,
     so a screen reader announces what to fix (WCAG 3.3.1). Each message links to its control by the
     Field's `id` (WCAG 2.4.3). FormPage carries the same summary; ItemPage and StepPage use this.

       <ErrorSummary errors={[{ field: 'email', message: 'Enter an email address' }]} /> -->
<script lang="ts">
	let { errors }: { errors: Array<{ field?: string; message: string }> } = $props();

	let summary: HTMLElement | undefined = $state();

	$effect(() => {
		if (errors.length > 0) summary?.focus();
	});
</script>

{#if errors.length > 0}
	<div class="summary" role="alert" tabindex="-1" bind:this={summary}>
		<h2>There is a problem</h2>
		<ul>
			{#each errors as error, i (i)}
				<li>{#if error.field}<a href="#{error.field}">{error.message}</a>{:else}{error.message}{/if}</li>
			{/each}
		</ul>
	</div>
{/if}

<style>
	.summary {
		max-width: 40rem;
		margin-bottom: var(--space-6);
		padding: var(--space-4);
		border: calc(var(--border-width) * 2) solid var(--destructive);
		border-radius: var(--radius);
	}
	.summary:focus-visible {
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}
	.summary h2 {
		margin: 0 0 0.5rem;
		font-size: var(--text-heading);
	}
	.summary ul {
		margin: 0;
		padding-inline-start: 1.25rem;
	}
</style>
