// Managed by Albitor (kit-carbon). Vitest reads this file instead of vite.config.ts, so it starts
// from that config and adds what the kit's tests need:
//   - Carbon ships .svelte files, which Node cannot load, so Vitest compiles them with the app;
//   - the `browser` condition, so `mount` resolves to Svelte's client build for the dialog test
//     (kit.test.ts renders on the server through `svelte/server`, which that does not change);
//   - src/lib/kit/vitest.setup.ts, which stubs the scrollIntoView that jsdom lacks and Carbon's SideNav calls.
import { defineConfig, mergeConfig } from 'vitest/config';
import viteConfig from './vite.config';

export default mergeConfig(
	viteConfig,
	defineConfig({
		resolve: { conditions: ['browser'] },
		test: { setupFiles: ['./src/lib/kit/vitest.setup.ts'], server: { deps: { inline: ['carbon-components-svelte'] } } }
	})
);
