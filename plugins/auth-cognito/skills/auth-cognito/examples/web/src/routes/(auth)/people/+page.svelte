<!-- The administrator's People screen for an
     invite-only app: who has an account, add someone, disable or enable them, reset a password.
     Nothing is emailed: adding or resetting shows a temporary password ONCE, which the
     administrator passes on; the person chooses their own at their first sign-in. Keep the
     data-testid values. In an open app (people sign themselves up) the page says so. -->
<script lang="ts">
	import { onMount, tick } from 'svelte';
	import {
		disablePerson,
		enablePerson,
		getSignUpMode,
		invitePerson,
		listPeople,
		resetPersonPassword,
		type Person,
		type SignUpMode,
		type TemporaryPassword
	} from '../../../lib/auth/client';
	import { getSession } from '../../../lib/auth/session';
	import { Button, Field } from '../../../lib/kit';

	let mode = $state<SignUpMode | null>(null);
	let me = $state('');
	let people = $state<Person[]>([]);
	let more = $state('');
	let loading = $state(true);
	let forbidden = $state(false);
	let error = $state('');
	let busy = $state(false);

	let email = $state('');
	let makeAdmin = $state(false);
	// The one-time reveal: held in memory until the administrator dismisses it.
	let revealed = $state<(TemporaryPassword & { reason: 'added' | 'reset' }) | null>(null);
	let copied = $state(false);
	let confirmReset = $state('');
	let revealBox: HTMLElement | undefined = $state();
	let alertBox: HTMLElement | undefined = $state();

	const enabledAdmins = $derived(people.filter((p) => p.admin && p.enabled).length);
	// Administrators here are the user-admin group's members. Only a complete list can say there is
	// a single one.
	const soleAdmin = $derived(more === '' && enabledAdmins === 1);

	async function fail(message: string) {
		error = message;
		await tick();
		alertBox?.focus();
	}

	async function load(append = false) {
		loading = true;
		const result = await listPeople(append ? more : undefined);
		loading = false;
		if (!result.ok) {
			if (result.code === 'forbidden') forbidden = true;
			else void fail(result.message);
			return;
		}
		people = append ? [...people, ...result.people] : result.people;
		more = result.next ?? '';
	}

	onMount(async () => {
		mode = await getSignUpMode();
		if (mode !== 'invite_only') {
			loading = false;
			return;
		}
		me = ((await getSession())?.email ?? '').toLowerCase();
		await load();
	});

	async function reveal(t: TemporaryPassword, reason: 'added' | 'reset') {
		revealed = { ...t, reason };
		copied = false;
		await tick();
		revealBox?.focus();
	}

	async function add(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		busy = true;
		const result = await invitePerson(email, makeAdmin);
		busy = false;
		if (!result.ok) {
			void fail(result.message);
			return;
		}
		email = '';
		makeAdmin = false;
		await reveal(result, 'added');
		await load();
	}

	async function toggle(p: Person) {
		error = '';
		busy = true;
		const result = await (p.enabled ? disablePerson(p.email) : enablePerson(p.email));
		busy = false;
		if (!result.ok) {
			void fail(result.message);
			return;
		}
		await load();
	}

	async function resetPassword(p: Person) {
		error = '';
		confirmReset = '';
		busy = true;
		const result = await resetPersonPassword(p.email);
		busy = false;
		if (!result.ok) {
			void fail(result.message);
			return;
		}
		await reveal(result, 'reset');
	}

	async function copy() {
		if (!revealed) return;
		try {
			await navigator.clipboard.writeText(revealed.temporary_password);
			copied = true;
		} catch {
			void fail('Could not copy: select the password and copy it yourself.');
		}
	}

	function label(p: Person): string {
		if (!p.enabled) return 'Disabled';
		return p.status === 'FORCE_CHANGE_PASSWORD' ? 'Invited' : 'Active';
	}
</script>

<svelte:head>
	<title>People</title>
</svelte:head>

<h1>People</h1>

{#if mode === null}
	<p role="status">Loading.</p>
{:else if mode !== 'invite_only'}
	<p data-testid="people-open">
		Anyone can create an account in this app, so there is no list of people to manage here.
	</p>
{:else if forbidden}
	<p data-testid="people-forbidden">Only an administrator can manage people.</p>
{:else}
	<p>
		People can only sign in once an administrator adds them here. Nothing is emailed: you will be
		shown a temporary password to pass on yourself.
	</p>

	{#if soleAdmin}
		<p role="note" data-testid="people-sole-admin">
			You are the only administrator. Add a second one, so the app is not locked out if you lose
			access to your account.
		</p>
	{/if}

	{#if revealed}
		<section
			aria-labelledby="people-reveal-heading"
			tabindex="-1"
			bind:this={revealBox}
			data-testid="people-reveal"
		>
			<h2 id="people-reveal-heading">
				{revealed.reason === 'added' ? 'Added' : 'Password reset for'}
				{revealed.email}
			</h2>
			<p>
				Their temporary password is shown below. <strong>It is shown only once</strong>: copy it now
				and pass it on securely. They will be asked to choose a password of their own when they first
				sign in.
			</p>
			<p><code data-testid="people-temp-password">{revealed.temporary_password}</code></p>
			<Button onclick={copy} variant="secondary" data-testid="people-copy">Copy password</Button>
			{#if copied}<span role="status" data-testid="people-copied">Copied.</span>{/if}
			<Button onclick={() => (revealed = null)} data-testid="people-dismiss">
				I have copied it
			</Button>
		</section>
	{/if}

	<h2>Add a person</h2>
	<form data-testid="people-add-form" onsubmit={add} novalidate>
		<Field
			label="Email"
			type="email"
			autocomplete="off"
			required
			bind:value={email}
			data-testid="people-email"
		/>
		<p>
			<label>
				<input type="checkbox" bind:checked={makeAdmin} data-testid="people-admin" />
				Make them an administrator
			</label>
		</p>
		<Button type="submit" disabled={busy} data-testid="people-add-button">Add person</Button>
	</form>

	{#if error}
		<p role="alert" tabindex="-1" bind:this={alertBox} data-testid="people-error">{error}</p>
	{/if}

	<h2>Everyone</h2>
	{#if loading && people.length === 0}
		<p role="status">Loading.</p>
	{:else}
		<table data-testid="people-table">
			<caption>People with an account</caption>
			<thead>
				<tr>
					<th scope="col">Email</th>
					<th scope="col">Role</th>
					<th scope="col">Status</th>
					<th scope="col">Actions</th>
				</tr>
			</thead>
			<tbody>
				{#each people as p (p.email)}
					<tr data-testid="people-row">
						<th scope="row">{p.email}</th>
						<td>{p.admin ? 'Administrator' : 'Member'}</td>
						<td>{label(p)}</td>
						<td>
							{#if p.email.toLowerCase() === me}
								<span>This is you</span>
							{:else}
								<Button
									variant="secondary"
									disabled={busy}
									onclick={() => toggle(p)}
									aria-label={`${p.enabled ? 'Disable' : 'Enable'} ${p.email}`}
								>
									{p.enabled ? 'Disable' : 'Enable'}
								</Button>
								{#if confirmReset === p.email}
									<Button
										variant="destructive"
										disabled={busy}
										onclick={() => resetPassword(p)}
										aria-label={`Confirm reset of the password for ${p.email}`}
									>
										Confirm reset
									</Button>
									<Button variant="ghost" onclick={() => (confirmReset = '')}>Cancel</Button>
								{:else}
									<Button
										variant="secondary"
										disabled={busy}
										onclick={() => (confirmReset = p.email)}
										aria-label={`Reset the password for ${p.email}`}
									>
										Reset password
									</Button>
								{/if}
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
		{#if more}
			<Button variant="secondary" disabled={loading} onclick={() => load(true)}>Show more</Button>
		{/if}
	{/if}
{/if}
