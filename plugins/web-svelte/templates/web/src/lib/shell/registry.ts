// A registry assembled by glob: panes, tabs, widgets, settings sections, anything a feature adds
// to a shared list. Each entry is its own small module beside the feature that owns it, and
// `import.meta.glob` collects them, so a feature adds a file and never edits a shared list (the
// way nav.ts entries already work). See the svelte-shell skill, "Add to a registry".

/** What every registry entry carries: an integer 0 to 1000, lower first. */
export interface Ordered {
	order: number;
}

/** A collected entry with the module path it came from, which is its stable identity. */
export type Collected<T> = T & { source: string };

/**
 * Collects the `exportName` export of every module `import.meta.glob` returned, validated by
 * `parse` (a value, or a reason it is unusable), ordered by `order` then module path. An unusable
 * entry throws in development, naming its file, and is skipped with a warning in a production
 * build, so one bad entry never blanks the app.
 */
export function collectRegistry<T extends Ordered>(
	modules: Record<string, unknown>,
	exportName: string,
	parse: (value: unknown) => T | string,
	dev: boolean = import.meta.env.DEV
): Collected<T>[] {
	const out: Collected<T>[] = [];
	const reject = (source: string, why: string) => {
		const message = `${exportName} entry ${source}: ${why}`;
		if (dev) throw new Error(message);
		console.warn(message);
	};
	for (const source of Object.keys(modules).sort()) {
		const parsed = parse((modules[source] as Record<string, unknown> | null)?.[exportName]);
		if (typeof parsed === 'string') {
			reject(source, parsed);
			continue;
		}
		if (!Number.isInteger(parsed.order) || parsed.order < 0 || parsed.order > 1000) {
			reject(source, 'order must be an integer from 0 to 1000');
			continue;
		}
		out.push({ ...parsed, source });
	}
	return out.sort((a, b) => a.order - b.order || a.source.localeCompare(b.source));
}
