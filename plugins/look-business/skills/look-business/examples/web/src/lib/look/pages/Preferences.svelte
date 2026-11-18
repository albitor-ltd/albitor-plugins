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

	let limit = $state('5000');
	let email = $state('');
	let reminders = $state(true);
	const error = (field: string) => errors.find((e) => e.field === field)?.message;
	async function save() {
		// Call the API here.
		toast.push({ message: 'Settings saved', tone: 'success' });
	}
</script>

<SettingsPage title="Settings" {errors} onsubmit={save}>
	{#snippet sections()}
		<SettingsSection title="Approvals" description="Invoices above this amount need a second approver.">
			<Field id="limit" label="Approval limit in pounds" inputmode="decimal" bind:value={limit} error={error('limit')} />
		</SettingsSection>
		<SettingsSection title="Reminders" description="Who is told when an invoice is overdue.">
			<Field id="email" label="Email address" type="email" autocomplete="email" bind:value={email} error={error('email')} />
			<label class="check">
				<input type="checkbox" bind:checked={reminders} />
				Send a reminder when an invoice is overdue
			</label>
		</SettingsSection>
	{/snippet}
</SettingsPage>

<style>
	.check {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-height: var(--control-height);
	}
</style>
