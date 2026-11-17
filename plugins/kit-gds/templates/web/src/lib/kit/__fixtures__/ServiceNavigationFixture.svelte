<!-- Test support: ServiceNavigation in the shell, for servicenavigation.test.ts alone. Delete it with ServiceNavigation.svelte and servicenavigation.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import ServiceNavigation from '../ServiceNavigation.svelte';

	let { path = '/reports/1' }: { path?: string } = $props();
</script>

<ServiceNavigation
	serviceName="Apply for a licence"
	items={[
		{ label: 'Home', href: '/' },
		{ label: 'Reports', href: '/reports' }
	]}
	{path}
/>
<main id="main">
	<h1 class="govuk-heading-l">Reports</h1>
</main>
