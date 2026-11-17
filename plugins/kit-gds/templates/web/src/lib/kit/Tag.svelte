<!-- A starting point, then yours: the GOV.UK tag (Frontend 6.4.0 markup), the status of a thing:
     "In progress", "Not started". One or two words, in the default style: no colour variants, so
     the status never depends on colour. It is a label, not a link or a button. -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();
</script>

<strong class="govuk-tag">{@render children()}</strong>
