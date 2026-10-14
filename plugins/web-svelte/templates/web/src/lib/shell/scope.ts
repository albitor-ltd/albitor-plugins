// Managed by Albitor (web-svelte): the names the generated slot files resolve.
//
// A plugin that fills web.providers or web.route-guard names an identifier, and ships a module in
// scope.d/ that exports it. Every module there is merged into one record, so no plugin edits a
// shared file. The generated providers file destructures the names it was given from this record.
const modules = import.meta.glob('./scope.d/*.ts', { eager: true }) as Record<
	string,
	Record<string, unknown>
>;

export function mergeScope(
	all: Record<string, Record<string, unknown>>
): Record<string, unknown> {
	const merged: Record<string, unknown> = {};
	const owner = new Map<string, string>();
	for (const [file, exports] of Object.entries(all)) {
		for (const [name, value] of Object.entries(exports)) {
			const first = owner.get(name);
			if (first !== undefined) {
				throw new Error(
					`scope.d export ${JSON.stringify(name)} is defined by both ${first} and ${file}`
				);
			}
			owner.set(name, file);
			merged[name] = value;
		}
	}
	return merged;
}

export const scope: Record<string, unknown> = mergeScope(modules);
