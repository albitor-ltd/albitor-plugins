// Managed by Albitor (web-svelte): the names the generated slot files resolve.
//
// A plugin that fills web.providers or web.route-guard names an identifier, and ships a module in
// scope.d/ that exports it. Every module there is merged into one record, so no plugin edits a
// shared file. The generated providers file destructures the names it was given from this record.
const modules = import.meta.glob('./scope.d/*.ts', { eager: true }) as Record<
	string,
	Record<string, unknown>
>;

export const scope: Record<string, unknown> = Object.assign(
	{},
	...Object.values(modules)
);
