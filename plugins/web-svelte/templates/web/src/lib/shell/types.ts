// Starting point (web-svelte): the shell's contract types.
import type { Component, Snippet } from 'svelte';

/** What a route's nav.ts exports as `nav`, and what a plugin's web.nav fragment carries. */
export interface NavMeta {
	/** Plain text, 1 to 64 characters: letters, digits, spaces and . , & ( ) / _ : - */
	label: string;
	/** An integer 0 to 1000; lower comes first. */
	order: number;
	/** A lower-case identifier; entries sharing one are listed together. */
	group?: string;
}

/** A nav entry resolved to a link. */
export interface NavItem extends NavMeta {
	href: string;
}

/** The nav as the kit's AppShell receives it: ungrouped entries first, then each group. */
export interface NavGroup {
	id: string | null;
	items: NavItem[];
}

/** A provider wraps the whole app. It renders its children and nothing else is required. */
export type ProviderComponent = Component<{ children: Snippet }>;

/**
 * A route guard runs before every navigation. Return a same-origin path to redirect there, or
 * nothing to allow it. It must allow the path it redirects to, or the app loops. A guard shapes
 * what the browser shows; the API must enforce access itself.
 */
export type Guard = (ctx: { url: URL }) => string | void | Promise<string | void>;
