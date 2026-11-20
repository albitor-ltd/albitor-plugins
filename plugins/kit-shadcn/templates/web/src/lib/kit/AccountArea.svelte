<!-- A starting point, then yours: the signed-in visitor's controls, in the one place the shell gives
     them. Render it inside AppShell's `account` snippet, never in a page body, a footer or a
     floating corner:

       {#snippet account()}
         <AccountArea name={user.name} theme={theme} onthemechange={setTheme}
                      saveStatus={status} onsignout={signOut} />
       {/snippet}
       <AppShell {nav} {path} {account}>...</AppShell>

     Each control draws only when its prop is given, so an app with no theme switch or no autosave
     shows neither. `saveStatus` is a polite live region ('saving', 'saved', 'error', or 'idle' for
     nothing); its text is always drawn, so a screen reader and a sighted user both get the words.
     `onthemechange` receives the theme the user asked for. Every control is a 44px target; on a
     phone the labels collapse to icons and keep their accessible names. It sits in AppShell's header
     row, which stays one row on a phone beside the Menu button. -->
<script module lang="ts">
	export type SaveStatus = 'idle' | 'saving' | 'saved' | 'error';
</script>

<script lang="ts">
	import type { Component } from 'svelte';
	import Check from '@lucide/svelte/icons/check';
	import LoaderCircle from '@lucide/svelte/icons/loader-circle';
	import LogOut from '@lucide/svelte/icons/log-out';
	import Moon from '@lucide/svelte/icons/moon';
	import Sun from '@lucide/svelte/icons/sun';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import Icon from './Icon.svelte';

	type Glyph = Component<{ size?: number | string; class?: string }>;

	let {
		name,
		theme,
		onthemechange,
		saveStatus = 'idle',
		onsignout
	}: {
		/** The signed-in person's display name, drawn before the controls. */
		name?: string;
		/** The current theme; give it with `onthemechange` to draw the switch. */
		theme?: 'light' | 'dark';
		onthemechange?: (next: 'light' | 'dark') => void;
		saveStatus?: SaveStatus;
		/** Draws Sign out. */
		onsignout?: () => void;
	} = $props();

	const words: Record<Exclude<SaveStatus, 'idle'>, string> = {
		saving: 'Saving',
		saved: 'Saved',
		error: 'Not saved'
	};
	const glyphs: Record<Exclude<SaveStatus, 'idle'>, Glyph> = {
		saving: LoaderCircle,
		saved: Check,
		error: TriangleAlert
	};
</script>

<div class="account-area" data-kit="account-area" role="group" aria-label="Account">
	<!-- The live region is always in the DOM, so a change is announced rather than a new node. -->
	<span class="status" data-status={saveStatus} role="status">
		{#if saveStatus !== 'idle'}
			<Icon icon={glyphs[saveStatus]} size={16} />
			<span>{words[saveStatus]}</span>
		{/if}
	</span>
	{#if name}<span class="name">{name}</span>{/if}
	{#if theme && onthemechange}
		<button
			type="button"
			class="control"
			aria-pressed={theme === 'dark'}
			onclick={() => onthemechange(theme === 'dark' ? 'light' : 'dark')}
		>
			<Icon icon={theme === 'dark' ? Moon : Sun} size={18} />
			<span class="label">Dark theme</span>
		</button>
	{/if}
	{#if onsignout}
		<button type="button" class="control" onclick={onsignout}>
			<Icon icon={LogOut} size={18} />
			<span class="label">Sign out</span>
		</button>
	{/if}
</div>

<style>
	.account-area {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		color: var(--header-ink);
	}
	.name {
		font-size: var(--text-small);
	}
	.status {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--text-small);
	}
	.status:empty {
		display: none;
	}
	.status[data-status='saving'] :global(.kit-icon) {
		animation: spin 1s linear infinite;
	}
	@media (prefers-reduced-motion: reduce) {
		.status[data-status='saving'] :global(.kit-icon) {
			animation: none;
		}
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
	.control {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
		min-height: 2.75rem;
		min-inline-size: 2.75rem;
		box-sizing: border-box;
		padding: var(--space-2) var(--space-3);
		font: inherit;
		color: var(--header-ink);
		background: transparent;
		border: var(--border-width) solid var(--border);
		border-radius: var(--radius);
		cursor: pointer;
		transition: background-color var(--motion-duration) var(--motion-ease);
	}
	.control:hover {
		background: var(--muted);
	}
	.control[aria-pressed='true'] {
		background: var(--second-subtle);
		color: var(--second-text);
	}
	.control:focus-visible {
		outline: var(--focus-width) solid var(--ring);
		outline-offset: 2px;
	}
	/* Phone: the name and the control labels give way to icons (the names stay for assistive
	   technology); the save status keeps its words, they are the only signal. */
	@media (width < 48rem) {
		.name {
			display: none;
		}
		.label {
			position: absolute;
			inline-size: 1px;
			block-size: 1px;
			margin: -1px;
			overflow: hidden;
			clip-path: inset(50%);
			white-space: nowrap;
		}
	}
</style>
