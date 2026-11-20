# Components

Every component is a file in `web/src/lib/kit/`, exported from `index.ts`. The kit is deliberately
small: the shell pieces a first page needs. For anything else, build on a Bits UI primitive
(`import { Select } from 'bits-ui'`), which brings the keyboard and ARIA behaviour, and style it
with the tokens. Browse https://bits-ui.com/docs/components for the primitives, and
https://www.shadcn-svelte.com/docs/components for the shadcn-svelte designs to copy from.

| Component | Props | Notes |
|---|---|---|
| `AppShell` | `nav: NavGroup[]`, `path: string`, `title?`, `mark?`, `footer?`, `account?`, `layout?` (`top`, `side`), `phoneNav?` (`drawer`, `tabs`), `icons?` (`{ [href]: LucideIcon }`), `children` | Rendered by web-svelte's root layout. Skip link, header, the nav (`NavShell`, only when `nav` has destinations), one `main`, the `Toaster` |
| `AccountArea` | `name?`, `theme?` + `onthemechange?`, `saveStatus?` (`idle`, `saving`, `saved`, `error`), `onsignout?` | Goes in `AppShell`'s `account` snippet, the one designed place for Sign out, the theme switch and the save status (desktop and phone; 44px targets, labels collapse to icons on a phone). Never put these in a page body, footer or floating corner |
| `NavShell` | `nav`, `path`, `open?`, `phoneNav?`, `side?`, `icons?` | Rendered by `AppShell`; do not render a second. Desktop: a row or, with `side`, a column. Phone: `drawer` (Menu button), or `tabs`: a bottom bar for 2 to 5 destinations, 4 tabs plus More for 6 or more, and the drawer for one |
| `Button` | `variant?` (`primary`, `secondary`, `destructive`, `ghost`), any `<button>` attribute | `type` defaults to `button`, so it never submits a form by accident |
| `Field` | `label`, `value` (bindable), `hint?`, `error?`, `multiline?`, `rows?`, any `<input>` attribute | Generates its own id; `error` sets `aria-invalid` |
| `Table` | `caption`, `columns: {key, label, priority?}[]`, `rows`, `narrow?` (`scroll`, `stacked`, `priority`), `emptyTitle?`, `emptyDescription?` | The first column is the row header. Empty rows render an `EmptyState`. `narrow` applies under 40rem of table width; see SKILL.md "Tables on a phone" |
| `Dialog` | `open` (bindable), `title`, `description`, `children?`, `actions?` | Bits UI `Dialog`; adds a Close button |
| `Toaster`, `toast` | `toast.push({ message, tone? })`, `dismiss(id)`, `clear()` | At most 5 shown, message 1 to 500 characters, no auto-dismiss |
| `EmptyState` | `title`, `description?`, `level?` (2 to 4), `icon?`, `action?` | Say why it is empty and what to do; with `icon` it is a first-run screen |
| `Icon` | `icon` (a Lucide component), `label?`, `size?` | `aria-hidden` unless `label` is set. Import icons from `@lucide/svelte/icons/<name>`, never the barrel |
| `Card` | `title?`, `level?` (2 to 4), `tone?` (`default`, `second`), `actions?`, `children` | With a title, a `group` named by its heading (not a landmark); elevation, border and radius from the look |
| `Pill` | `tone?` (`neutral`, `success`, `warning`, `danger`, `info`), `children` | A glyph plus the word; filled with the tone's ink and fill pair |
| `Tabs` | `label`, `tabs: {value, label}[]`, `value?` (bindable), `panel` snippet | Bits UI `Tabs`: arrow keys and Home/End, the panel tied to its tab. For views of one thing, not page navigation |
| `Avatar` | `name`, `src?`, `size?` (`sm`, `md`, `lg`) | Bits UI `Avatar`; initials when there is no photo or it fails; named for the person |
| `Progress` | `label`, `value` (or `null`), `max?`, `variant?` (`bar`, `ring`), `showValue?` | Bits UI `Progress`: a named `progressbar`; `null` is indeterminate |
| `Skeleton` | `width?`, `height?`, `circle?` | Decorative (`aria-hidden`); put it in an `aria-busy` region with a loading message. Shimmer follows the look's motion |
| `FilterChip` | `pressed` (bindable), `children` | A toggle button: `aria-pressed`, plus a check when on |
| `StatTile` | `label`, `value`, `icon?`, `delta?`, `meter?`, `spark?` | A `dl` of label and values; the change is words, with an arrow as decoration |
| `Chart` | `title`, `data: {label, value}[]`, `kind?` (`bar`, `sparkline`), `compact?`, `unit?` | SVG in the tokens, plus a visually hidden table of the values. A handful of values, not a charting library |

## Extending

- A new variant: add the class in `Button.svelte`'s `<style>` using tokens, add it to the
  `variant` union.
- A table cell that needs markup (a link, a badge): copy `Table.svelte` and render a snippet for
  that column. Keep `{...}` text interpolation for everything that is user data.
- Keep the Bits UI wiring and the `data-` and `aria-` attributes it sets when you restyle.
