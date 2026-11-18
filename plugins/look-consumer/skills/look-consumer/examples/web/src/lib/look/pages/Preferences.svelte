<!-- A settings page, in this look: a few titled groups, each its own roomy Card, one Save for the
     page. A starting point, then yours: replace the groups and the fields, and confirm a save
     with a toast. Needs the kit's SettingsPage and SettingsSection (copy them from the kit-shadcn
     skill). -->
<script lang="ts">
	import Field from '../../kit/Field.svelte';
	import SettingsPage from '../../kit/pages/SettingsPage.svelte';
	import SettingsSection from '../../kit/pages/SettingsSection.svelte';
	import { toast } from '../../kit/toast.svelte';

	let { errors = [] }: { errors?: Array<{ field?: string; message: string }> } = $props();

	let name = $state('Sam');
	let time = $state('19:30');
	let weekly = $state(true);
	const error = (field: string) => errors.find((e) => e.field === field)?.message;
	async function save() {
		// Call the API here.
		toast.push({ message: 'Settings saved', tone: 'success' });
	}
</script>

<SettingsPage title="Settings" {errors} onsubmit={save}>
	{#snippet sections()}
		<SettingsSection title="About you">
			<Field id="name" label="First name" autocomplete="given-name" bind:value={name} error={error('name')} />
		</SettingsSection>
		<SettingsSection title="Reminders" description="When to nudge you about habits still to do.">
			<Field id="time" label="Reminder time" hint="Use 24-hour time, for example 19:30." bind:value={time} error={error('time')} />
			<label class="check">
				<input type="checkbox" bind:checked={weekly} />
				Email me a summary every Sunday
			</label>
		</SettingsSection>
	{/snippet}
</SettingsPage>

<style>
	.check {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		min-height: var(--control-height);
	}
</style>
