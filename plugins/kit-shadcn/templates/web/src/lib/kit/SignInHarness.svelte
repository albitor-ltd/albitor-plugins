<!-- Managed by Albitor (kit-shadcn): drives the kit's SignInPage through its states the way an auth
     plugin does (error and busy changing over time), for signin.test.ts. Not routed. -->
<script lang="ts">
	import { SignInPage } from './index';

	let view = $state<{ error?: string; email?: string; busy: boolean }>({ busy: false });

	/** Change the props the page is given, as the owning route would between attempts. */
	export function set(next: { error?: string; email?: string; busy?: boolean }): void {
		view = { ...view, ...next };
	}
</script>

<SignInPage email={view.email} error={view.error} busy={view.busy} onsubmit={() => {}} />
