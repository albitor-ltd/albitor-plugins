<!-- A first-run screen, in this look: nothing created yet. Small, calm art above one plain
     sentence that says why the page is empty and the one action that fixes it. In a dense table
     or a multi-column admin view leave the art out and keep the sentence and the action. Use it
     on a first-run page only, never for an error or an empty search. A starting point, then
     yours: replace the title, the sentence and the actions. -->
<script lang="ts">
	import Button from '../../kit/Button.svelte';
	import PageHeader from '../../kit/pages/PageHeader.svelte';
	import EmptyArt from '../EmptyArt.svelte';

	const noop = () => {};
</script>

<PageHeader title="Invoices" />

<section class="first-run" aria-labelledby="first-run-title">
	<EmptyArt />
	<h2 id="first-run-title">No invoices yet</h2>
	<p>Add the first invoice, or import a file of them, and they are listed here for approval.</p>
	<div class="actions">
		<Button onclick={noop}>New invoice</Button>
		<Button variant="secondary" onclick={noop}>Import a file</Button>
	</div>
</section>

<style>
	.first-run {
		display: grid;
		justify-items: center;
		gap: var(--space-3);
		padding: var(--space-8) var(--space-4);
		text-align: center;
	}
	h2 {
		margin: 0;
		font-size: var(--text-heading);
	}
	p {
		margin: 0;
		max-width: 44ch;
		color: var(--muted-foreground);
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: var(--space-2);
	}
</style>
