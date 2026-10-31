// Starting point (kit-gds). Vitest reads this file instead of vite.config.ts, so it starts
// from that config and adds what the kit's tests need: the `browser` condition, so `mount`
// resolves to Svelte's client build for the dialog test (kit.test.ts renders on the server
// through `svelte/server`, which that does not change).
import { defineConfig, mergeConfig } from 'vitest/config';
import viteConfig from './vite.config';

export default mergeConfig(viteConfig, defineConfig({ resolve: { conditions: ['browser'] } }));
