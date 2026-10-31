// Starting point (web-svelte): checks and runs what the web.providers and web.route-guard
// slots resolved to.
import type { Guard, ProviderComponent } from './types';

/**
 * A slot names an identifier that the plugin's scope.d module must export. An unresolved name
 * would otherwise surface as `undefined is not a function` deep in rendering, so fail at load.
 */
function resolved<T>(slot: string, values: readonly unknown[]): T[] {
	values.forEach((value, i) => {
		if (typeof value !== 'function') {
			throw new Error(
				`${slot}[${i}] did not resolve: a scope.d module must export the name the plugin declared`
			);
		}
	});
	return values as T[];
}

export function checkProviders(values: readonly unknown[]): ProviderComponent[] {
	return resolved<ProviderComponent>('web.providers', values);
}

export function checkGuards(values: readonly unknown[]): Guard[] {
	return resolved<Guard>('web.route-guard', values);
}

/** A redirect target must stay on this origin: one path, no scheme, host or backslash tricks. */
export function isSafeRedirect(to: string): boolean {
	return /^\/(?![/\\])[^\\\u0000-\u001f]*$/.test(to);
}

/** Runs the guards in order and returns the first redirect, or null to allow the navigation. */
export async function runGuards(guards: readonly Guard[], url: URL): Promise<string | null> {
	for (const guard of guards) {
		const to = await guard({ url });
		if (to === undefined || to === null) continue;
		if (typeof to !== 'string' || !isSafeRedirect(to)) {
			throw new Error('a route guard returned a redirect that is not a same-origin path');
		}
		return to;
	}
	return null;
}
