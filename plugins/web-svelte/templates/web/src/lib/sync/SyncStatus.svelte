<!-- Starting point (web-svelte, offline data layer): a polite status line for the sync state. Place it in the
     layout or a page; it renders nothing until someone is signed in. -->
<script lang="ts">
	import { getSync } from './context';
	import { liveStatus } from './live.svelte';
	import { statusText } from './status';

	const view = liveStatus(getSync());
	const line = $derived(statusText(view.status, view.online));
</script>

<p role="status" class="sync-status" data-tone={line.tone} hidden={line.text === ''}>{line.text}</p>
