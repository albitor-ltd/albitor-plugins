<!-- Shown when the sign-up domain rule (SIGNUP_DOMAINS)
     refuses the address a person signed in with. The API sends them here and sets no session. Reword
     it for the service, and say who to contact. -->
<svelte:head>
	<title>You cannot use this service with that email address</title>
</svelte:head>

<h1>You cannot use this service with that email address</h1>
<p>
	This service is only open to people with an email address from an approved organisation.
	The address you signed in with is not one of them.
</p>
<p>If you think this is wrong, contact the team that runs this service.</p>
<!-- A POST form, never a link: a link to sign-out could sign a visitor out from another site. It
     goes through Cognito's sign-out page, which clears Cognito's own cookie; without that, the
     next "Sign in" would reuse the same address. -->
<form method="POST" action="/api/auth/sign-out" data-testid="not-allowed-sign-out-form">
	<button type="submit" data-testid="not-allowed-sign-out">Sign out and use a different email address</button>
</form>
