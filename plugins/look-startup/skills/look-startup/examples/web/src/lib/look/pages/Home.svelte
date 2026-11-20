<!-- The public home page of a landing-led product, in this look: a confident headline, one plain
     sentence, one primary action, then reasons as layered cards (hairline plus a soft shadow) each
     with a Lucide icon, a short numbered "how it works" and a closing prompt. No imagery: the
     type scale and the teal accent on the action do the work. Second colour on the quiet
     highlight only. A starting point, then yours: replace the copy and the reasons. Built on the
     kit's HomePage (copy the kit's page templates from the kit-shadcn skill). -->
<script lang="ts">
	import Gauge from '@lucide/svelte/icons/gauge';
	import GitBranch from '@lucide/svelte/icons/git-branch';
	import ShieldCheck from '@lucide/svelte/icons/shield-check';
	import HomePage from '../../kit/pages/HomePage.svelte';
</script>

<HomePage
	title="Ship the change, not the pipeline"
	lead="Connect a repository and every merge is built, tested and live in under a minute. No scripts to maintain."
	primary={{ href: '/trial', label: 'Start a free trial' }}
	secondary={{ href: '/releases/2026-11', label: 'See what changed this month' }}
	highlights={[
		{ title: 'Fast by default', text: 'Median build time is 58 seconds, shown on every deploy.', icon: Gauge },
		{ title: 'Works with your branches', text: 'Preview each branch on its own address before it merges.', icon: GitBranch },
		{ title: 'Safe to roll back', text: 'One click returns to the last good release and says what it changed.', icon: ShieldCheck }
	]}
	closing={{
		title: 'Try it on one project',
		text: 'No card needed. Cancel by deleting the project.',
		action: { href: '/trial', label: 'Start a free trial' }
	}}
>
	<section>
		<h2>How it works</h2>
		<ol>
			<li>Connect a repository.</li>
			<li>Push a change.</li>
			<li>Watch it go live, and share the preview address.</li>
		</ol>
	</section>
</HomePage>

<style>
	h2 {
		margin: 0 0 var(--space-3);
		font-size: var(--text-heading);
	}
	ol {
		display: grid;
		gap: var(--space-2);
		margin: 0;
		padding-inline-start: 1.25rem;
	}
</style>
