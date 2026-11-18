<!-- A form page, in this look: a short form in one Card, labels above the controls, one named
     primary action and a plain Cancel. A starting point, then yours: replace the fields and the
     validation, and save in `onsubmit`. Validate on the server too. Needs the kit's FormPage
     (copy it from the kit-shadcn skill). -->
<script lang="ts">
	import Field from '../../kit/Field.svelte';
	import FormPage from '../../kit/pages/FormPage.svelte';

	let { errors = [] }: { errors?: Array<{ field?: string; message: string }> } = $props();

	let name = $state('');
	let repo = $state('');
	const nameError = $derived(errors.find((e) => e.field === 'name')?.message);
	const repoError = $derived(errors.find((e) => e.field === 'repo')?.message);
	async function save() {
		// Call the API here, then go to the new project.
	}
</script>

<FormPage
	title="New project"
	description="A project is one thing you deploy."
	submitLabel="Create project"
	cancelHref="/projects"
	{errors}
	onsubmit={save}
>
	<Field id="name" label="Project name" hint="Lowercase letters, numbers and hyphens." bind:value={name} error={nameError} />
	<Field id="repo" label="Repository" hint="For example example/api-gateway." bind:value={repo} error={repoError} />
</FormPage>
