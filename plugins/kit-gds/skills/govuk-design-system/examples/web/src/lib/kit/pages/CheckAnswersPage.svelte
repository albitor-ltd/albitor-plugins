<!-- Step 3: check answers. Every answer in a summary list, each with a Change link that returns to
     that question page (and names it in hidden text). One primary button sends; its name says what
     it does ("Accept and send"), and a busy guard stops a double submit. The page after it is the
     confirmation.

       <CheckAnswersPage rows={rows} backHref="/apply" onsubmit={send} /> -->
<script lang="ts">
	import BackLink from '../BackLink.svelte';
	import Button from '../Button.svelte';
	import SummaryList from '../SummaryList.svelte';
	import type { SummaryRow } from '../SummaryList.svelte';

	let {
		title = 'Check your answers before sending your application',
		rows,
		backHref,
		submitLabel = 'Accept and send',
		busy = false,
		onsubmit
	}: {
		title?: string;
		rows: SummaryRow[];
		backHref: string;
		submitLabel?: string;
		busy?: boolean;
		onsubmit: () => void | Promise<void>;
	} = $props();

	function handle(event: SubmitEvent) {
		event.preventDefault();
		if (!busy) void onsubmit();
	}
</script>

<BackLink href={backHref} />

<div class="govuk-grid-row">
	<div class="govuk-grid-column-two-thirds">
		<h1 class="govuk-heading-l">{title}</h1>
		<SummaryList {rows} />
		<form onsubmit={handle}>
			<Button type="submit" disabled={busy}>{busy ? 'Sending…' : submitLabel}</Button>
		</form>
	</div>
</div>
