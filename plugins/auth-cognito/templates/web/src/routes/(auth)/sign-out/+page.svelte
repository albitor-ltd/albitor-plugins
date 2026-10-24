<!-- Seeded by Albitor (auth-cognito, component-kit@2), then yours: signs the person out and sends them to
     the sign-in screen. Its nav.ts puts "Sign out" in the nav; delete nav.ts to take it out. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { signOut } from '../../../lib/auth/client';

	onMount(async () => {
		await signOut();
		await goto('/sign-in');
	});
</script>

<svelte:head>
	<title>Signing out</title>
</svelte:head>

<h1>Signing out</h1>
<p role="status">One moment.</p>
