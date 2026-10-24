<!-- Seeded by Albitor (kit-gds), then yours: a sign-in screen (component-kit@2 `SignInPage`) in
     GOV.UK markup. It is presentation only: the identity plugin (auth@1) decides what signing in
     means, so call its client from `onsubmit` and pass back a plain `error` sentence. Say "Email
     or password is incorrect" and never which one, keep the password field
     `autocomplete="current-password"` so a password manager works, and do not block paste (WCAG
     3.3.8 Accessible Authentication). The page has its own <h1>. If the auth plugin already ships
     sign-in routes, restyle those with these pieces rather than adding a second route.

       <SignInPage {error} {busy} onsubmit={(email, password) => signIn(email, password)}
                   resetHref="/reset" signUpHref="/sign-up" />

     A failure shows GOV.UK's error summary (focused, linked to the email field) above the heading;
     the fields carry no message of their own, because the failure is about the pair. -->
<script lang="ts">
	import { untrack } from 'svelte';
	import Button from './Button.svelte';
	import ErrorSummary from './ErrorSummary.svelte';
	import Field from './Field.svelte';

	let {
		title = 'Sign in',
		error,
		busy = false,
		resetHref,
		signUpHref,
		email: initialEmail = '',
		onsubmit
	}: {
		title?: string;
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

	// The error stays in the page until the next attempt's result replaces it. The summary takes
	// focus when it mounts, so it is keyed to remount, and refocus, only for a failed attempt, a
	// repeated sentence included: when `busy` falls with an error set, or a new error arrives
	// while idle. An error that changes mid-attempt waits for `busy` to fall.
	// Worked out while rendering, not in an effect, so the first error mounts the summary once.
	let attempt = 0;
	let wasBusy = false;
	let shown: string | undefined;
	const summaryKey = $derived.by(() => {
		const settled = wasBusy && !busy;
		const changed = error !== shown;
		wasBusy = busy;
		shown = error;
		if (error && !busy && (settled || changed)) attempt += 1;
		return attempt;
	});

	function handle(event: SubmitEvent) {
		event.preventDefault();
		if (!busy) void onsubmit(email, password);
	}
</script>

<div class="kit-signin">
	{#if error}
		{#key summaryKey}
			<ErrorSummary errors={[{ field: 'signin-email', message: error }]} />
		{/key}
	{/if}
	<h1 class="govuk-heading-l">{title}</h1>
	<form onsubmit={handle} novalidate>
		<Field
			id="signin-email"
			label="Email address"
			type="email"
			bind:value={email}
			autocomplete="username"
			spellcheck={false}
			required
		/>
		<Field
			id="signin-password"
			label="Password"
			type="password"
			bind:value={password}
			autocomplete="current-password"
			required
		/>
		<Button type="submit" disabled={busy}>{busy ? 'Signing in…' : 'Sign in'}</Button>
	</form>
	{#if resetHref || signUpHref}
		<p class="govuk-body kit-signin__links">
			{#if resetHref}<a class="govuk-link" href={resetHref}>Forgot your password?</a>{/if}
			{#if signUpHref}<a class="govuk-link" href={signUpHref}>Create an account</a>{/if}
		</p>
	{/if}
</div>

<style>
	.kit-signin {
		max-width: 30rem;
	}
	.kit-signin__links {
		display: flex;
		flex-direction: column;
		gap: 10px;
		margin-top: 20px;
	}
</style>
