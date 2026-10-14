# Components

Every component is a file in `web/src/lib/kit/`, exported from `index.ts`. The kit is deliberately
small: the shell pieces a first page needs. For anything else, build on a Bits UI primitive
(`import { Select } from 'bits-ui'`), which brings the keyboard and ARIA behaviour, and style it
with the tokens. Browse https://bits-ui.com/docs/components for the primitives, and
https://www.shadcn-svelte.com/docs/components for the shadcn-svelte designs to copy from.

| Component | Props | Notes |
|---|---|---|
| `AppShell` | `nav: NavGroup[]`, `path: string`, `title?`, `children` | Rendered by web-svelte's root layout. Skip link, header, labelled nav, one `main`, the `Toaster` |
| `Button` | `variant?` (`primary`, `secondary`, `destructive`, `ghost`), any `<button>` attribute | `type` defaults to `button`, so it never submits a form by accident |
| `Field` | `label`, `value` (bindable), `hint?`, `error?`, `multiline?`, `rows?`, any `<input>` attribute | Generates its own id; `error` sets `aria-invalid` |
| `Table` | `caption`, `columns: {key, label}[]`, `rows`, `emptyTitle?`, `emptyDescription?` | The first column is the row header. Empty rows render an `EmptyState` |
| `Dialog` | `open` (bindable), `title`, `description`, `children?`, `actions?` | Bits UI `Dialog`; adds a Close button |
| `Toaster`, `toast` | `toast.push({ message, tone? })`, `dismiss(id)`, `clear()` | At most 5 shown, message 1 to 500 characters, no auto-dismiss |
| `EmptyState` | `title`, `description?`, `level?` (2 to 4), `action?` | Say why it is empty and what to do |

## Extending

- A new variant: add the class in `Button.svelte`'s `<style>` using tokens, add it to the
  `variant` union.
- A table cell that needs markup (a link, a badge): copy `Table.svelte` and render a snippet for
  that column. Keep `{...}` text interpolation for everything that is user data.
- Keep the Bits UI wiring and the `data-` and `aria-` attributes it sets when you restyle.
