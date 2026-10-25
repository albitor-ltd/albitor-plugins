<!-- Seeded by Albitor (auth-hosted, component-kit@2), then yours: the person's account, and the way
     to delete it. Its nav.ts puts "Account" in the nav; delete nav.ts to take it out. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { getSession } from '../../../lib/auth/session';
	import { Button } from '../../../lib/kit';

	let email = $state('');

	onMount(async () => {
		email = (await getSession())?.email ?? '';
	});
</script>

<svelte:head>
	<title>Your account</title>
</svelte:head>

<h1>Your account</h1>
{#if email}<p data-testid="account-email">Email address: {email}</p>{/if}
<p>To change your password, sign out and use "Forgot your password?" on the sign-in page.</p>

<h2>Delete your account</h2>
<p>You will be asked to confirm before anything is deleted.</p>
<Button variant="destructive" onclick={() => goto('/account/delete')} data-testid="account-delete">
	Delete account
</Button>
