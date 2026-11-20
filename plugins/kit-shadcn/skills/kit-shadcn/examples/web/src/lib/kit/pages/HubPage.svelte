<!-- A portal hub: the first screen of an app with no bar of destinations. A returning person finds
     tiles to each thing they can do, each with a line of status ("Due 1 December", "Nothing to do"),
     and a section for what needs them now. The tiles are the navigation, so every destination
     appears here and every other page keeps a link back (BackToHub, or `back` on PageHeader): with
     `nav={[]}` the shell has no nav landmark and nothing is hidden on a phone. The tile's title is
     the link text; the whole tile is the target.

       <HubPage title="Your account" description="Everything you can do, in one place."
                tiles={[{ href: '/renew', title: 'Renew your permit', text: 'Takes about 5 minutes.',
                          icon: RefreshCw, status: { tone: 'warning', label: 'Due 1 December' } }]} />

     `children` takes sections below the tiles (recent activity, messages), each with its own <h2>. -->
<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import Icon from '../Icon.svelte';
	import Pill from '../Pill.svelte';
	import PageHeader from './PageHeader.svelte';
	import StateNotice from './StateNotice.svelte';

	type Glyph = Component<{ size?: number | string; class?: string }>;

	let {
		title,
		description,
		tilesTitle = 'What do you want to do?',
		tiles,
		status = 'ready',
		errorMessage,
		onretry,
		actions,
		children
	}: {
		title: string;
		description?: string;
		tilesTitle?: string;
		tiles: Array<{
			href: string;
			title: string;
			text: string;
			icon?: Glyph;
			/** Where this task stands: a Pill, always a glyph and a word. */
			status?: { tone: 'neutral' | 'success' | 'warning' | 'danger' | 'info'; label: string };
		}>;
		status?: 'loading' | 'error' | 'ready';
		errorMessage?: string;
		onretry?: () => void;
		actions?: Snippet;
		children?: Snippet;
	} = $props();
</script>

<PageHeader {title} {description} {actions} />

{#if status !== 'ready'}
	<StateNotice {status} {errorMessage} {onretry} loadingLabel="Loading your account" />
{:else}
	<h2 class="tiles-title">{tilesTitle}</h2>
	<ul class="tiles">
		{#each tiles as tile (tile.href)}
			<li>
				<span class="chip">{#if tile.icon}<Icon icon={tile.icon} size={22} />{/if}</span>
				<div class="body">
					<h3><a href={tile.href}>{tile.title}</a></h3>
					<p>{tile.text}</p>
					{#if tile.status}<p class="state"><Pill tone={tile.status.tone}>{tile.status.label}</Pill></p>{/if}
				</div>
			</li>
		{/each}
	</ul>
	{#if children}<div class="more">{@render children()}</div>{/if}
{/if}

<style>
	.tiles-title {
		margin: 0 0 var(--space-3);
		font-size: var(--text-heading);
	}
	.tiles {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(16rem, 1fr));
		gap: var(--space-4);
		margin: 0;
		padding: 0;
		list-style: none;
	}
	li {
		position: relative;
		display: flex;
		gap: var(--space-3);
		min-height: 5.5rem;
		padding: var(--space-4);
		background: var(--surface);
		border: var(--border-width) solid var(--border);
		border-radius: var(--radius);
		box-shadow: var(--shadow);
	}
	li:hover {
		border-color: var(--input);
	}
	li:focus-within {
		outline: var(--focus-width) solid var(--ring);
		outline-offset: 2px;
	}
	.chip {
		display: inline-flex;
		flex: none;
		align-items: center;
		justify-content: center;
		width: 2.5rem;
		height: 2.5rem;
		border-radius: var(--radius);
		background: var(--second-subtle);
		color: var(--second-text);
	}
	.chip:empty {
		display: none;
	}
	h3 {
		margin: 0 0 var(--space-1);
		font-size: var(--text-body);
	}
	h3 a {
		color: var(--link);
		text-decoration: none;
	}
	h3 a:hover {
		text-decoration: underline;
	}
	h3 a::after {
		content: '';
		position: absolute;
		inset: 0;
	}
	h3 a:focus-visible {
		outline: none;
	}
	.body p {
		margin: 0;
		color: var(--muted-foreground);
		font-size: var(--text-small);
	}
	.body .state {
		margin-top: var(--space-2);
	}
	.more {
		margin-top: var(--space-8);
	}
	.more :global(h2) {
		font-size: var(--text-heading);
	}
</style>
