<!-- A starting point, then yours: GOV.UK inset text (Frontend 6.4.0 markup). Text set apart from the
     body, such as a deadline or a rule the person needs to see; use it sparingly, once or twice a
     page at most. Never for a warning or an error (use a notification banner or the error summary). -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();
</script>

<div class="govuk-inset-text">{@render children()}</div>
