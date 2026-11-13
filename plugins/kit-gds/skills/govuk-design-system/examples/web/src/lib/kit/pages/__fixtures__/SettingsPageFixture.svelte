<!-- Test support: SettingsPage in the shell, for settingspage.test.ts alone. Delete it with SettingsPage.svelte and SettingsSection.svelte
     and settingspage.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import Field from '../../Field.svelte';
	import SettingsPage from '../SettingsPage.svelte';
	import SettingsSection from '../SettingsSection.svelte';
	import PagesShowcase from './PagesShowcase.svelte';

	let {
		errors = [],
		busy = false
	}: {
		errors?: Array<{ field?: string; message: string }>;
		busy?: boolean;
	} = $props();

	let name = $state('');
	const noop = () => {};
</script>

<PagesShowcase>
	<SettingsPage title="Settings" {errors} {busy} onsubmit={noop}>
		{#snippet sections()}
			<SettingsSection title="Profile" description="How you appear to your team.">
				<Field
					id="display-name"
					label="Display name"
					bind:value={name}
					error={errors.length ? 'Enter a display name' : undefined}
				/>
			</SettingsSection>
			<SettingsSection title="Notifications">
				<div class="govuk-form-group">
					<fieldset class="govuk-fieldset">
						<legend class="govuk-fieldset__legend govuk-fieldset__legend--s">Email me about</legend>
						<div class="govuk-checkboxes govuk-checkboxes--small">
							<div class="govuk-checkboxes__item">
								<input class="govuk-checkboxes__input" id="notify-reports" name="notify" type="checkbox" />
								<label class="govuk-label govuk-checkboxes__label" for="notify-reports">New reports</label>
							</div>
						</div>
					</fieldset>
				</div>
			</SettingsSection>
		{/snippet}
	</SettingsPage>
</PagesShowcase>
