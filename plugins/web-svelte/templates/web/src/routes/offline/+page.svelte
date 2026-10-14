<!-- Seeded by Albitor (web-svelte), then yours: where the app sends people it cannot reach. -->
<svelte:head>
	<title>You are offline</title>
</svelte:head>

<h1>You are offline</h1>
<p>This page needs a connection. Check your network, then try again.</p>
<p><button type="button" onclick={() => location.reload()}>Try again</button></p>
