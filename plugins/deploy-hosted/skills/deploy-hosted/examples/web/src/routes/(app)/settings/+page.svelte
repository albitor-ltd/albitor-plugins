<!-- Settings: the landing page for the app's settings. Notifications is linked here, and it is
     also in the nav through its own nav.ts. Add the app's other settings pages to the list. A
     link to /settings/notifications must stay: a check of the finished app looks for it. -->
<svelte:head>
	<title>Settings</title>
</svelte:head>

<h1>Settings</h1>
<ul>
	<li>
		<a href="/settings/notifications" data-testid="settings-notifications">Notifications</a>
		<span>: which emails you get</span>
	</li>
</ul>
