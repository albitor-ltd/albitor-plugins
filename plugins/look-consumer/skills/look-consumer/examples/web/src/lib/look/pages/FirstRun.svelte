<!-- A first-run screen, in this look: nothing created yet. The look's round, soft art above one
     plain sentence that says why the page is empty and the one action that fixes it, with the
     action large and centred where a thumb reaches. Use it on a first-run page only, never for an
     error or an empty search. A starting point, then yours: replace the title, the sentence and
     the action. -->
<script lang="ts">
	import Button from '../../kit/Button.svelte';
	import PageHeader from '../../kit/pages/PageHeader.svelte';
	import EmptyArt from '../EmptyArt.svelte';

	const noop = () => {};
</script>

<PageHeader title="Habits" />

<section class="first-run" aria-labelledby="first-run-title">
	<EmptyArt />
	<h2 id="first-run-title">No habits yet</h2>
	<p>Add your first habit and your streak starts the day you do it.</p>
	<Button onclick={noop}>Add a habit</Button>
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
		margin: 0 0 var(--space-3);
		max-width: 32ch;
		color: var(--muted-foreground);
	}
</style>
