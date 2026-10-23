<!-- Seeded by Albitor (kit-carbon), then yours: a sign-in screen. It is presentation only: the
     identity plugin (auth@1) decides what signing in means, so call its client from `onsubmit` and
     pass back a plain `error` sentence. Say "Email or password is incorrect" and never which one,
     keep the password field `autocomplete="current-password"` so a password manager works, and do
     not block paste (WCAG 3.3.8 Accessible Authentication). The page has its own <h1>. If the auth
     plugin already ships sign-in routes, restyle those with these pieces rather than adding a
     second route.

       <SignInPage {error} {busy} onsubmit={(email, password) => signIn(email, password)}
                   resetHref="/reset" signUpHref="/sign-up" /> -->
<script lang="ts">
	import { untrack } from 'svelte';
	import Button from '../Button.svelte';
	import Field from '../Field.svelte';

	let {
		title = 'Sign in',
		error,
		busy = false,
		resetHref,
		signUpHref,
		onsubmit
	}: {
		title?: string;
		/** One plain sentence; never which of email or password was wrong. */
		error?: string;
		busy?: boolean;
		resetHref?: string;
		signUpHref?: string;
		onsubmit: (email: string, password: string) => void | Promise<void>;
	} = $props();

	let email = $state('');
	let password = $state('');
	let alertBox: HTMLElement | undefined = $state();

	// The error stays in the page until the next attempt's result replaces it, and takes focus on
	// every failed attempt, a repeated sentence included: when the error changes while idle, or when
	// `busy` falls with an error set.
	let wasBusy = false;
	let shown: string | undefined;
	$effect(() => {
		const settled = wasBusy && !busy;
		const changed = error !== shown;
		wasBusy = busy;
		shown = error;
		if (error && !busy && (settled || changed)) untrack(() => alertBox)?.focus();
	});

	function handle(event: SubmitEvent) {
		event.preventDefault();
		if (!busy) void onsubmit(email, password);
	}
</script>

<div class="card">
	<h1>{title}</h1>
	{#if error}
		<p class="error" role="alert" tabindex="-1" bind:this={alertBox}>
			<span class="sr">Error:</span> {error}
		</p>
	{/if}
	<form onsubmit={handle} novalidate>
		<Field label="Email address" type="email" bind:value={email} autocomplete="username" required />
		<Field
			label="Password"
			type="password"
			bind:value={password}
			autocomplete="current-password"
			required
		/>
		<Button type="submit" disabled={busy}>{busy ? 'Signing in…' : 'Sign in'}</Button>
	</form>
	{#if resetHref || signUpHref}
		<p class="links">
			{#if resetHref}<a href={resetHref}>Forgot your password?</a>{/if}
			{#if signUpHref}<a href={signUpHref}>Create an account</a>{/if}
		</p>
	{/if}
</div>

<style>
	.card {
		max-width: 24rem;
		margin: 2rem auto;
	}
	h1 {
		margin: 0 0 1rem;
		font-size: 2rem;
	}
	.error {
		margin: 0 0 1rem;
		padding: 0.75rem 1rem;
		border: 2px solid var(--cds-text-error);
		border-radius: 0;
		color: var(--cds-text-error);
		font-weight: 600;
	}
	.error:focus-visible {
		outline: 2px solid var(--cds-focus);
		outline-offset: 2px;
	}
	.sr {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}
	.links {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		margin-top: 1rem;
	}
	.links a {
		color: var(--cds-link-primary);
	}
</style>
