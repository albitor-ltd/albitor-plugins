<!-- A starting point, then yours: a sign-in screen. It is presentation only: the
     app's identity code decides what signing in means, so call its client from `onsubmit` and
     pass back a plain `error` sentence. Say "Email or password is incorrect" and never which one,
     keep the password field `autocomplete="current-password"` so a password manager works, and do
     not block paste (WCAG 3.3.8 Accessible Authentication). The page has its own <h1>. If the app
     already ships sign-in routes, restyle those with these pieces rather than adding a
     second route.

       <SignInPage {error} {busy} onsubmit={(email, password) => signIn(email, password)}
                   resetHref="/reset" signUpHref="/sign-up" /> -->
<script lang="ts">
	import { untrack } from 'svelte';
	import Button from './Button.svelte';
	import Field from './Field.svelte';

	let {
		title = 'Sign in',
		mark,
		error,
		busy = false,
		resetHref,
		signUpHref,
		email: initialEmail = '',
		onsubmit
	}: {
		title?: string;
		/** URL of the app's mark (`/brand/mark.svg`), shown above the heading with an empty alt. */
		mark?: string;
		/** One plain sentence; never which of email or password was wrong. */
		error?: string;
		busy?: boolean;
		resetHref?: string;
		signUpHref?: string;
		/** The email field's initial value, for a person returning from sign-up or confirm. */
		email?: string;
		onsubmit: (email: string, password: string) => void | Promise<void>;
	} = $props();

	// An initial value only: the person's typing wins, and a later prop change does not overwrite it.
	let email = $state(untrack(() => initialEmail));
	let password = $state('');
	// Hidden when the file is missing, so an app with no platform-written brand files shows no broken image.
	let markFailed = $state(false);
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
	{#if mark && !markFailed}<img class="mark" src={mark} alt="" onerror={() => (markFailed = true)} />{/if}
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
		margin: var(--space-8) auto;
	}
	.mark {
		display: block;
		height: 3rem;
		width: auto;
		max-width: 6rem;
		margin: 0 0 var(--space-4);
		object-fit: contain;
	}
	h1 {
		margin: 0 0 var(--space-4);
		font-size: var(--text-title);
	}
	.error {
		margin: 0 0 var(--space-4);
		padding: var(--space-3) var(--space-4);
		border: calc(var(--border-width) + 1px) solid var(--destructive);
		border-radius: var(--radius);
		color: var(--destructive);
		font-weight: 600;
	}
	.error:focus-visible {
		outline: var(--focus-width) solid var(--ring);
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
		gap: var(--space-2);
		margin-top: var(--space-4);
	}
	.links a {
		color: var(--link);
	}
</style>
