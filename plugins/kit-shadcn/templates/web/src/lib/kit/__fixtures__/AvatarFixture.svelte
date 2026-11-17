<!-- Test support: Avatar in the shell, for avatar.test.ts alone. Delete it with Avatar.svelte and
     avatar.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import AppShell from '../AppShell.svelte';
	import Avatar from '../Avatar.svelte';
	import { fixtureNav } from './nav';
</script>

<AppShell nav={fixtureNav()} path="/" title="Example app">
	<h1>People</h1>
	<Avatar name="Ada Lovelace" />
	<Avatar name="Grace" size="sm" />
</AppShell>
