<!-- Test support: SignInPage in the pages frame, for pages.test.ts. SignInPage is part of the shell
     contract, so this fixture and that test stay. Not routed; nothing links here. -->
<script lang="ts">
	import { SignInPage } from '../../index';
	import PagesShowcase from './PagesShowcase.svelte';

	let { signInError }: { signInError?: string } = $props();
</script>

<PagesShowcase>
	<SignInPage error={signInError} resetHref="/reset" signUpHref="/sign-up" onsubmit={() => {}} />
</PagesShowcase>
