// Managed by Albitor (web-svelte): turns route nav.ts modules and plugin web.nav entries into the
// grouped, ordered list the kit's AppShell renders.
import type { NavGroup, NavItem, NavMeta } from './types';

const LABEL = /^[A-Za-z0-9 .,&()/_:-]{1,64}$/;
const GROUP = /^[a-z0-9][a-z0-9_-]{0,63}$/;
const ROUTES_PREFIX = '/src/routes';
const NAV_SUFFIX = '/nav.ts';

/** Validates a `nav` export, returning it or a reason it is unusable. */
export function parseNavMeta(value: unknown): NavMeta | string {
	if (typeof value !== 'object' || value === null) return 'nav must be an object';
	const { label, order, group, ...rest } = value as Record<string, unknown>;
	const extra = Object.keys(rest);
	if (extra.length > 0) return `unknown field ${JSON.stringify(extra[0])}`;
	if (typeof label !== 'string' || !LABEL.test(label)) {
		return 'label must be 1 to 64 letters, digits, spaces or . , & ( ) / _ : -';
	}
	if (typeof order !== 'number' || !Number.isInteger(order) || order < 0 || order > 1000) {
		return 'order must be an integer from 0 to 1000';
	}
	if (group !== undefined && (typeof group !== 'string' || !GROUP.test(group))) {
		return 'group must be a lower-case identifier';
	}
	return group === undefined ? { label, order } : { label, order, group };
}

/**
 * The link for the route whose nav.ts sits at `file` (an import.meta.glob key such as
 * `/src/routes/(app)/reports/nav.ts`). Route groups vanish from the path. A route with a
 * parameter has no single link, so it returns null and cannot be a nav entry.
 */
export function hrefForNavFile(file: string): string | null {
	if (!file.startsWith(ROUTES_PREFIX + '/') || !file.endsWith(NAV_SUFFIX)) return null;
	const dir = file.slice(ROUTES_PREFIX.length, file.length - NAV_SUFFIX.length);
	const segments = dir.split('/').filter((s) => s !== '' && !/^\(.*\)$/.test(s));
	if (segments.some((s) => s.includes('[') || s.includes(']') || s === '..' || s === '.')) {
		return null;
	}
	return '/' + segments.join('/');
}

/**
 * A plugin's web.nav entry has no target, only a label. Its link is the label lower-cased with
 * each run of other characters turned into one hyphen: "Sign in" links to /sign-in.
 */
export function hrefForLabel(label: string): string {
	const slug = label
		.toLowerCase()
		.replace(/[^a-z0-9]+/g, '-')
		.replace(/^-+|-+$/g, '');
	if (slug === '') {
		throw new RangeError(`nav label ${JSON.stringify(label)} has no letters or digits to link to`);
	}
	return '/' + slug;
}

function compareItems(a: NavItem, b: NavItem): number {
	return a.order - b.order || a.label.localeCompare(b.label) || a.href.localeCompare(b.href);
}

/**
 * Merges plugin entries with the modules `import.meta.glob` collected. A route and a plugin
 * naming one link keep the route's entry, so an app can restyle what a plugin offered. An
 * unusable entry throws in development and is skipped in a production build, so one bad nav.ts
 * never blanks the app.
 */
export function buildNav(
	pluginNav: readonly NavMeta[],
	routeModules: Record<string, unknown>,
	dev: boolean = import.meta.env.DEV
): NavGroup[] {
	const byHref = new Map<string, NavItem>();
	const reject = (where: string, why: string) => {
		const message = `nav entry ${where}: ${why}`;
		if (dev) throw new Error(message);
		console.warn(message);
	};

	const pluginHrefs = new Set<string>();
	pluginNav.forEach((entry, i) => {
		const meta = parseNavMeta(entry);
		if (typeof meta === 'string') return reject(`web.nav[${i}]`, meta);
		let href: string;
		try {
			href = hrefForLabel(meta.label);
		} catch (err) {
			return reject(`web.nav[${i}]`, (err as Error).message);
		}
		if (pluginHrefs.has(href)) return reject(`web.nav[${i}]`, `duplicate link ${href}`);
		pluginHrefs.add(href);
		byHref.set(href, { ...meta, href });
	});

	for (const file of Object.keys(routeModules).sort()) {
		const href = hrefForNavFile(file);
		if (href === null) {
			reject(file, 'a route with a parameter or an unusual path cannot be a nav entry');
			continue;
		}
		const exported = (routeModules[file] as { nav?: unknown } | null)?.nav;
		const meta = parseNavMeta(exported);
		if (typeof meta === 'string') {
			reject(file, meta);
			continue;
		}
		byHref.set(href, { ...meta, href });
	}

	const groups = new Map<string | null, NavItem[]>();
	for (const item of byHref.values()) {
		const id = item.group ?? null;
		groups.set(id, [...(groups.get(id) ?? []), item]);
	}
	return [...groups.entries()]
		.sort(([a], [b]) => (a === b ? 0 : a === null ? -1 : b === null ? 1 : a.localeCompare(b)))
		.map(([id, items]) => ({ id, items: items.sort(compareItems) }));
}
