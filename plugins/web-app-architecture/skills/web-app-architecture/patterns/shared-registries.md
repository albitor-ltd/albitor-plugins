# shared-registries: features add files, never lines to a shared list

Nodes built in parallel that each append to one hand-kept list (a pane or tab table, a route
table, a barrel `index.ts`, a settings-section array, a kit component, `.gitignore`) conflict on
every merge. The mechanical merge resolves additive single-line entries and ignore-file patterns;
an edit inside an array or object literal, or to a shared component's markup, goes to an agent and
holds the build while it does. Remove the shared edit instead of merging it.

## The rules

- **A list features extend is assembled by glob.** Each entry is its own module in the feature's
  directory (`pane.ts` exporting `pane`, `tab.ts` exporting `tab`); one registry file collects them
  with `import.meta.glob`. The web-svelte shell seeds `collectRegistry` for this and its nav works
  the same way (`svelte-shell` skill, "Add to a registry"). A new pane is a new file.
- **The skeleton node writes the registry once.** The shared node that owns the shell creates the
  registry file and the page that renders it. Feature nodes only add entries, so none of them waits
  on another over the list.
- **Where a list is unavoidable it is append-only, one entry per line.** Append at the end; never
  reorder, reformat or touch another entry's line. A trailing comma on every line keeps the diff to
  one added line.
- **A feature never edits a shared kit component.** Wrap or extend it inside the feature's own
  directory (a `FeatureTable.svelte` that renders the kit `Table`); a change every feature needs
  belongs to the skeleton node, once.
- **Generated outputs are ignored by the seed.** A feature does not edit `.gitignore`; if an output
  is missing, add one line at the end.

## When it does not apply

A list one node owns outright (its own enum, its own route table) needs no registry. The rule is
for a list that two nodes of one plan could both add to.
