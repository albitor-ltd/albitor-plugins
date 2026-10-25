// Managed by Albitor (kit-carbon). jsdom has no scrollIntoView, and carbon-components-svelte 0.113's
// SideNav calls it on its active item when the shell mounts, which would be an unhandled rejection
// in any jsdom test that renders the app shell. Node-environment tests have no Element and skip it.
if (typeof Element !== 'undefined') {
	Element.prototype.scrollIntoView ??= () => {};
}
