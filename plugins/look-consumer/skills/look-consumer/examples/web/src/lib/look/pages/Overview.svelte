<!-- The home screen, in this look: a header band with the look's art, one focus card with a
     progress ring and the one thing to do next, then big friendly figure tiles and a short list.
     On a phone the main action floats above the tab bar, within thumb reach. A starting point,
     then yours: replace the copy, the figures and the list, and pass the load `status`. Built from
     StateNotice and StatTile (copy the kit's page templates from the kit-shadcn skill). Before the app has any data, show FirstRun.svelte instead. -->
<script lang="ts">
	import Flame from '@lucide/svelte/icons/flame';
	import ListChecks from '@lucide/svelte/icons/list-checks';
	import Plus from '@lucide/svelte/icons/plus';
	import Trophy from '@lucide/svelte/icons/trophy';
	import Button from '../../kit/Button.svelte';
	import Card from '../../kit/Card.svelte';
	import Icon from '../../kit/Icon.svelte';
	import Pill from '../../kit/Pill.svelte';
	import Progress from '../../kit/Progress.svelte';
	import StatTile from '../../kit/StatTile.svelte';
	import StateNotice from '../../kit/pages/StateNotice.svelte';
	import EmptyArt from '../EmptyArt.svelte';

	let { status = 'ready' }: { status?: 'loading' | 'error' | 'ready' } = $props();

	const next = [
		{ name: 'Evening walk', when: 'Due today', tone: 'warning', state: 'To do' },
		{ name: 'Read for 20 minutes', when: 'Done at 08:10', tone: 'success', state: 'Done' }
	] as const;
	const noop = () => {};
</script>

{#if status !== 'ready'}
	<h1 class="plain">Today</h1>
	<StateNotice {status} onretry={noop} loadingLabel="Loading today" />
{:else}
	<header class="band">
		<div>
			<h1>Good morning, Sam</h1>
			<p>Three of four habits done today. One to go.</p>
		</div>
		<span class="art"><EmptyArt size={104} /></span>
	</header>

	<section class="focus" aria-labelledby="focus-title">
		<Progress label="Habits done today" variant="ring" value={75} showValue />
		<div class="text">
			<p class="eyebrow">Up next</p>
			<h2 id="focus-title">Evening walk</h2>
			<p>Ten minutes outside counts.</p>
		</div>
		<Button onclick={noop}>Mark done</Button>
	</section>

	<div class="tiles">
		<StatTile label="Current streak" value="12 days" icon={Flame} delta={{ direction: 'up', text: '2 more than last week', good: true }} />
		<StatTile label="Habits" value={4} icon={ListChecks} />
		<StatTile label="Best streak" value="31 days" icon={Trophy} />
	</div>

	<Card title="Today">
		<ul class="list">
			{#each next as item (item.name)}
				<li>
					<span class="what"><b>{item.name}</b><small>{item.when}</small></span>
					<Pill tone={item.tone}>{item.state}</Pill>
				</li>
			{/each}
		</ul>
	</Card>

	<div class="fab"><Button onclick={noop}><Icon icon={Plus} size={20} /> Add habit</Button></div>
{/if}

<style>
	.band {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-4);
		padding: var(--space-6);
		margin-bottom: var(--space-6);
		border-radius: var(--radius);
		background: color-mix(in oklab, var(--primary) 11%, var(--surface));
	}
	h1,
	.plain {
		margin: 0;
		font-family: var(--heading-font);
		font-weight: var(--heading-weight);
		font-size: clamp(1.5rem, 5vw, var(--text-title));
		line-height: 1.15;
	}
	.band p {
		margin: var(--space-2) 0 0;
		color: var(--muted-foreground);
	}
	.art {
		flex: none;
	}
	.focus {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-4);
		padding: var(--space-6);
		margin-bottom: var(--space-6);
		border: var(--border-width) solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		box-shadow: var(--shadow);
	}
	.text {
		flex: 1;
		min-width: 10rem;
	}
	.text h2 {
		margin: 0;
		font-size: var(--text-heading);
	}
	.text p {
		margin: var(--space-1) 0 0;
	}
	.eyebrow {
		color: var(--second-text);
		font-size: var(--text-small);
		font-weight: 700;
	}
	.tiles {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
		gap: var(--space-4);
		margin-bottom: var(--space-6);
	}
	.list {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.list li {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3) 0;
		border-top: var(--border-width) solid var(--border);
	}
	.list li:first-child {
		border-top: 0;
	}
	.what {
		display: grid;
	}
	.what small {
		color: var(--muted-foreground);
		font-size: var(--text-small);
	}
	.fab {
		display: none;
	}
	@media (width < 48rem) {
		.art :global(svg) {
			inline-size: 5.25rem;
			block-size: 5.25rem;
		}
		.fab {
			display: block;
			position: fixed;
			inset-inline-end: var(--space-4);
			inset-block-end: calc(var(--tab-bar-offset, 0px) + var(--space-4));
			box-shadow: var(--shadow);
			border-radius: 999px;
		}
	}
</style>
