<!-- Managed by Albitor (kit-gds): the component-kit@2 shell contract on one page. AppShell, Footer
     and SignInPage are imported from the kit's index, the way an auth plugin imports them, so
     contract.test.ts fails if the kit stops exporting one. Managed so the audit cannot be deleted
     by editing it; it is not routed and nothing links here. -->
<script lang="ts">
	import { AppShell, Footer, SignInPage } from './index';
	import type { NavGroup } from '../shell/types';

	let {
		withFooter = true,
		error,
		busy = false,
		extraLink,
		repeatLink = false
	}: { withFooter?: boolean; error?: string; busy?: boolean; extraLink?: string; repeatLink?: boolean } = $props();

	const nav: NavGroup[] = [
		{ id: null, items: [{ label: 'Home', order: 0, href: '/' }] }
	];
</script>

{#snippet footer()}
	<Footer
		links={[
			{ href: '/privacy', label: 'Privacy' },
			{ href: '/accessibility', label: 'Accessibility statement' },
			...(repeatLink ? [{ href: '/privacy', label: 'Privacy again' }] : []),
			...(extraLink ? [{ href: '/extra', label: extraLink }] : [])
		]}
	>
		Example Ltd
</Footer>
{/snippet}

<AppShell {nav} path="/" title="Example app" footer={withFooter ? footer : undefined}>
	<SignInPage {error} {busy} resetHref="/reset" signUpHref="/sign-up" onsubmit={() => {}} />
</AppShell>
