import type { Guard } from '../types';

export { default as StubProvider } from '../../stub/StubProvider.svelte';

// Sends everyone off /private to /, and allows every other path.
export const stubGuard: Guard = ({ url }) => (url.pathname === '/private' ? '/' : undefined);
