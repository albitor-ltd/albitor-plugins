---
name: kit-carbon
description: How to build pages from the kit-carbon component kit on a SvelteKit app composed from web-svelte (IBM Carbon style, on carbon-components-svelte, in Carbon's white and Gray 100 themes). Use when a build adds or changes a page, form, table, dialog, toast or empty state under web/src, touches the Carbon theme, or touches web/src/lib/kit.
---

# kit-carbon: the component kit

The kit follows the ownership model of the other Svelte kit. The wrappers are **source files in
your app** (`web/src/lib/kit/`), seeded once and yours to edit. Underneath they use
[carbon-components-svelte](https://svelte.carbondesignsystem.com), which is a runtime package, and
Carbon's themes for colour, type and spacing. Carbon is **not themeable** by a visual-language
plugin: do not paste another token set over it.

Import from the barrel with a relative path (SvelteKit 3 has no `$lib` alias here):

```ts
import { Button, Field, Table, Dialog, EmptyState, toast } from '../../lib/kit';
```

## Rules

1. **Use a kit component if one exists.** A `<div>` modal or a `<span onclick>` button throws away
   the accessibility the kit carries. `references/components.md` lists what exists; for anything
   else use a Carbon component (`import { Select } from 'carbon-components-svelte'`).
2. **Style with Carbon's properties, never hex.** Use `var(--cds-text-primary)`,
   `var(--cds-layer)` and the rest (`references/tokens.md`). A hardcoded colour breaks the dark
   theme and the contrast audit.
3. **Label every control.** `Field` ties the label, hint and error to the control; never use a
   placeholder as the label.
4. **Name every icon-only control** with visible text or an `aria-label`. Icons come from
   `carbon-icons-svelte` (`import Add from 'carbon-icons-svelte/lib/Add.svelte'`); give a button an
   `icon` and keep its text.
5. **A dialog always has a title and a description.** `Dialog` requires both props.
6. **Values are text.** `Table` cells, toast messages and nav labels render as text. Never feed user
   input to `{@html}`.

## Pages

A page is `web/src/routes/<name>/+page.svelte` (see the `svelte-shell` skill for nav entries). Start
from `Showcase.svelte`, which is every piece on one page and what `kit.test.ts` audits.

| Need | Use |
|---|---|
| A form field | `<Field label hint error bind:value />`; `multiline` for a textarea; `type` for the input type |
| An action | `<Button variant="primary" \| "secondary" \| "destructive" \| "ghost">`; `type="submit"` in a form; `icon={Add}` for a Carbon icon |
| Tabular data | `<Table caption columns rows />`; an empty `rows` shows an `EmptyState` |
| Nothing to show | `<EmptyState title description>` with an `action` snippet saying what to do next |
| A confirmation or small task | `<Dialog bind:open title description>` with an `actions` snippet |
| Feedback after an action | `toast.push({ message, tone })` (`tone`: `info`, `success`, `error`) |

`AppShell` already renders the one `<Toaster />`; do not render a second. Toasts never disappear on
a timer and each can be dismissed.

The shell is Carbon's: a fixed header, a side nav on the left (a hamburger below 1056px), and the
page in `main`. Do not add a second nav. Pages sit inside Carbon's padding; use Carbon's `Grid`
(`Row`, `Column`) for layout, not a hand-rolled one.

Patterns (`references/patterns.md`) cover forms with validation, destructive confirmation, empty
and error states.

## Page templates

Start every new page from the template that matches it. The templates are example files in
`web/src/lib/kit/pages/` (import from `'../../lib/kit/pages'`): seeded once, then yours to edit.
They are built from the kit's own pieces, so the header, the loading and failed states, the error
summary and the labelling are already right and already audited (`pages/pages.test.ts` runs axe over
every template in every state, in addition to the contrast test). Do not rebuild them by hand.

| The page is | Start from | It gives you |
|---|---|---|
| A collection of records | `ListPage` | Header with actions, a filter group, `Table`, `EmptyState` with a next step, loading and failed states |
| One record | `DetailPage` | Back link, actions, facts as a description list, related sections, loading and failed states |
| Create or edit one record | `FormPage` | Labelled fields, a focused error summary, a named submit, cancel, a busy guard |
| An overview | `DashboardPage` | Labelled headline figures, sections below, loading and failed states |
| Preferences, in groups | `SettingsPage` with `SettingsSection` | Named groups, one save, the same error summary |
| Sign in | `SignInPage` | Email and password with the right `autocomplete`, one alert for the failure; presentation only |

`PageHeader` (the one `<h1>`, a description, a back link, actions) and `StateNotice` (loading
`status`, failure `alert` with a retry) are the skeleton under every template; use them directly for
a page that fits none of the above, such as a wizard step or an import report.

How to use one:

1. Copy the template's usage comment (the top of the file) into the route and fill in the props. A
   page is still `web/src/routes/<name>/+page.svelte`; the template is a component inside it.
2. Fetch in `+page.ts` or the page, and pass `status` (`loading`, `error`, `ready`) with `rows`,
   `fields` or `stats`. Give `onretry` so a failed load can be tried again.
3. Edit the template when the product needs something it lacks (a column with markup, a second
   action). Keep its headings in order (one `<h1>`, then `<h2>`), and keep every control labelled.
4. A destructive action opens a `Dialog`; it never fires on one click.
5. `SignInPage` is presentation. Call the identity plugin's client from `onsubmit` and pass back a
   plain `error` sentence. If the auth plugin already ships sign-in routes, restyle those with these
   pieces instead of adding a second sign-in route.

## Theming

There is none beyond the two themes. `AppShell` loads `carbon-components-svelte/css/all.css` and a
Carbon `Theme` sets `white` or `g100` on `<html>` from the visitor's preference; `<html
data-theme="light">` or `"dark"` forces one (`theme.ts`). The kit ships no token file and no
`albitor-kit` cascade layer, because it is not themeable by a visual-language plugin: such a plugin
cannot restyle it, and a build should not pair them.

To change the look, change the Carbon component or token you use, not the palette. If you need a
colour Carbon does not give you, you are probably using the wrong component. If you add a colour
pair to a component, add it to the `pairs` list in `contrast.test.ts` in the same change: that test
reads the installed Carbon stylesheet and computes WCAG 2.2 AA ratios (4.5:1 for text, 3:1 for a
control edge, the focus ring and the primary fill) in both themes. `kit.test.ts` renders the shell
pieces and runs axe on them; axe cannot judge contrast without a layout engine, which is why the
contrast test exists.

The general WCAG 2.2 AA content (criteria, audits, accessibility statements) lives in the
`test-playwright` plugin, not here.

## Accessibility built in

| Piece | What the kit already does |
|---|---|
| `AppShell` | Skip link first, one `main` landmark, a labelled `nav`, `aria-current="page"` on the current link (`true` inside its section), visible focus |
| `Field` | `label for`, `aria-describedby` to the hint and error, `aria-invalid`, the error announced as an alert |
| `Table` | A caption, `scope` on every header, a keyboard-focusable scroll region |
| `Dialog` | Named by its title, described by its description, Tab wraps inside it, Escape closes, focus returns (Carbon) |
| `Toaster` | Always-present `status` (polite) and `alert` (assertive, for errors) regions |

## Do not

- Do not use Carbon's `ToastNotification` directly for toasts: it sets its own alert role and puts
  the message in an `h3`. Use `toast.push`.
- Do not override Carbon's `.bx--*` classes to recolour; that is theming, which this kit does not
  support.
- Do not hand-roll focus management; Carbon's `Modal` already traps focus.
- Do not remove the focus outlines.
- Do not import Carbon's CSS a second time or pick a theme other than `white` and `g100`; the
  contrast audit covers those two.
