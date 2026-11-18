<!-- A settings page, in this look: titled groups, each its own Card, one Save for the page. A
     starting point, then yours: replace the groups and the fields, and confirm a save with a
     toast. Needs the kit's SettingsPage and SettingsSection (copy them from the kit-shadcn
     skill). -->
<script lang="ts">
	import Field from '../../kit/Field.svelte';
	import SettingsPage from '../../kit/pages/SettingsPage.svelte';
	import SettingsSection from '../../kit/pages/SettingsSection.svelte';
	import { toast } from '../../kit/toast.svelte';

	let { errors = [] }: { errors?: Array<{ field?: string; message: string }> } = $props();

	let team = $state('Platform');
	let email = $state('');
	let digest = $state('weekly');
	const emailError = $derived(errors.find((e) => e.field === 'email')?.message);
	async function save() {
		// Call the API here.
		toast.push({ message: 'Settings saved', tone: 'success' });
	}
</script>

<SettingsPage title="Settings" {errors} onsubmit={save}>
	{#snippet sections()}
		<SettingsSection title="Team" description="Shown beside every project you own.">
			<Field id="team" label="Team name" bind:value={team} />
		</SettingsSection>
		<SettingsSection title="Notifications" description="Where failed deployments are reported.">
			<Field id="email" label="Email address" type="email" autocomplete="email" bind:value={email} error={emailError} />
			<fieldset>
				<legend>Summary</legend>
				<label><input type="radio" name="digest" value="daily" bind:group={digest} /> Daily</label>
				<label><input type="radio" name="digest" value="weekly" bind:group={digest} /> Weekly</label>
			</fieldset>
		</SettingsSection>
	{/snippet}
</SettingsPage>

<style>
	fieldset {
		border: 0;
		margin: var(--space-4) 0 0;
		padding: 0;
		display: grid;
		gap: var(--space-2);
	}
	legend {
		font-weight: 600;
		margin-bottom: var(--space-2);
	}
	label {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-height: var(--control-height);
	}
</style>
