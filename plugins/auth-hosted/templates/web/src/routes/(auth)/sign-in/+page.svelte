<!-- Seeded by Albitor (auth-hosted), then yours. There is no form: people sign in on
     the app's Cognito hosted page. This page sends the browser there straight away (through the
     API, which starts the sign-in), and its button is the fallback. When the API sends a person back
     with ?error=<code> the sign-in did not finish: the page says so and does NOT redirect again, so a
     failure cannot loop, and a second arrival within 30 seconds does not redirect either. Restyle and reword freely. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { nextPath } from '../../../lib/auth/guard';
	import { loginUrl, signInErrorMessage } from '../../../lib/auth/client';
	import { Button } from '../../../lib/kit';

	const error = page.url.searchParams.get('error');
	const href = loginUrl(nextPath(page.url.searchParams.get('next')));

	// The page hands the browser to Cognito at most once in this window. A second arrival so soon
	// means the sign-in "worked" but no session was kept (a blocked cookie, say), and going round
	// again would loop for ever: say so instead. The stamp is per tab and best effort.
	const WINDOW_MS = 30_000;
	const STAMP = 'app.sign-in.at';
	let looped = $state(false);

	function go() {
		window.location.assign(href);
	}

	onMount(() => {
		if (error) return;
		try {
			const last = Number(sessionStorage.getItem(STAMP));
			if (last && Date.now() - last < WINDOW_MS) {
				looped = true;
				return;
			}
			sessionStorage.setItem(STAMP, String(Date.now()));
		} catch {
			// No storage: redirect, as the first arrival would.
		}
		window.location.replace(href);
	});
</script>

<svelte:head>
	<title>{error ? 'Sign-in did not finish' : looped ? 'Sign-in did not stick' : 'Continuing to sign in'}</title>
</svelte:head>

{#if error}
	<h1>We could not sign you in</h1>
	<p role="alert" data-testid="sign-in-error">{signInErrorMessage(error)}</p>
	<Button onclick={go} data-testid="sign-in-continue">Continue to sign in</Button>
{:else if looped}
	<h1>We could not keep you signed in</h1>
	<p role="alert" data-testid="sign-in-looped">
		You signed in, but this browser did not keep the session. Check that it accepts cookies from this site, then try again.
	</p>
	<Button onclick={go} data-testid="sign-in-continue">Continue to sign in</Button>
{:else}
	<h1>Continuing to sign in</h1>
	<p role="status">Taking you to the sign-in page. If nothing happens, use the button below.</p>
	<Button onclick={go} data-testid="sign-in-continue">Continue to sign in</Button>
{/if}
