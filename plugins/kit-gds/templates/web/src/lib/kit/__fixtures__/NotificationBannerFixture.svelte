<!-- Test support: NotificationBanner in the shell, for notificationbanner.test.ts alone. Delete it with NotificationBanner.svelte and notificationbanner.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import { createRawSnippet } from 'svelte';
	import AppShell from '../AppShell.svelte';
	import NotificationBanner from '../NotificationBanner.svelte';
	import { fixtureNav } from './nav';

	let {
		type = 'important',
		title,
		singleLine = false
	}: { type?: 'important' | 'success'; title?: string; singleLine?: boolean } = $props();

	const body = createRawSnippet(() => ({
		render: () => '<p class="govuk-notification-banner__heading">Your report was saved.</p>'
	}));
</script>

<AppShell nav={fixtureNav()} path="/reports" title="Example app">
	{#if singleLine}
		<NotificationBanner {type} {title} heading="Your report was saved." />
	{:else}
		<NotificationBanner {type} {title} children={body} />
	{/if}
	<h1 class="govuk-heading-l">Reports</h1>
</AppShell>
