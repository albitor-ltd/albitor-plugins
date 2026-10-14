# Components

Every component is a file in `web/src/lib/kit/`, exported from `index.ts`. The kit is deliberately
small: the shell pieces a first page needs. For anything else use a Carbon component directly
(`import { Select, Tabs } from 'carbon-components-svelte'`), which brings the keyboard and ARIA
behaviour, and check it with axe. Browse https://svelte.carbondesignsystem.com for the components
and https://carbondesignsystem.com for the guidance.

| Component | Props | Notes |
|---|---|---|
| `AppShell` | `nav: NavGroup[]`, `path: string`, `title?`, `children` | Rendered by web-svelte's root layout. Carbon `Header`, skip link, a labelled `SideNav`, one `main`, the `Toaster` |
| `Button` | `variant?` (`primary`, `secondary`, `destructive`, `ghost`), `icon?` (a Carbon icon), any `<button>` attribute | `type` defaults to `button`, so it never submits a form by accident. `destructive` is Carbon's `danger` |
| `Field` | `label`, `value` (bindable), `hint?`, `error?`, `multiline?`, `rows?`, any `<input>` attribute | Carbon `TextInput` / `TextArea`; generates its own id; `error` sets `aria-invalid`; the hint stays when there is an error |
| `Table` | `caption`, `columns: {key, label}[]`, `rows`, `emptyTitle?`, `emptyDescription?` | The first column is the row header. Empty rows render an `EmptyState`. For sorting or selection use Carbon's `DataTable` |
| `Dialog` | `open` (bindable), `title`, `description`, `children?`, `actions?` | Carbon `Modal` (passive, so the footer is yours: put buttons in `actions`). It stays in the page, inert, while closed |
| `Toaster`, `toast` | `toast.push({ message, tone? })`, `dismiss(id)`, `clear()` | At most 5 shown, message 1 to 500 characters, no auto-dismiss |
| `EmptyState` | `title`, `description?`, `level?` (2 to 4), `action?` | Say why it is empty and what to do |

## Extending

- A new variant: add it to `kinds` in `Button.svelte` and to the `variant` union (Carbon has
  `primary`, `secondary`, `tertiary`, `ghost`, `danger`, `danger-tertiary`, `danger-ghost`).
- A table cell that needs markup (a link, a tag): copy `Table.svelte` and render a snippet for that
  column. Keep `{...}` text interpolation for everything that is user data.
- A Carbon component is Svelte 4 style: it takes slots and `on:` events, and works inside this
  Svelte 5 app. Wrap it in a runes component, as the kit does, so the rest of the app sees plain props.
- Keep the `aria-` attributes Carbon sets when you restyle.
