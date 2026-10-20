---
name: kit-shadcn
description: How to build pages from the kit-shadcn component kit on a SvelteKit app composed from web-svelte (shadcn-style components on Bits UI, one token set), the design system the app's UI is built and reviewed against. Use when a build adds or changes a page, form, table, dialog, toast or empty state under web/src, themes the app, or touches web/src/lib/kit.
metadata:
  capability: design-system
---

# kit-shadcn: the component kit

The kit follows the shadcn ownership model. The components are **source files in your app**
(`web/src/lib/kit/`), seeded once and yours to edit; there is no runtime package to upgrade. The
behaviour that is hard to get right (focus trap, Escape, scroll lock, returning focus) comes from
[Bits UI](https://bits-ui.com), the headless Svelte primitives that shadcn-svelte is built on.
Colours, radius and type come from one token file.

Import from the barrel with a relative path (SvelteKit 3 has no `$lib` alias here):

```ts
import { Button, Field, Table, Dialog, EmptyState, toast } from '../../lib/kit';
```

## Rules

1. **Use a kit component if one exists.** A `<div>` modal or a `<span onclick>` button throws away
   the accessibility the kit carries. `references/components.md` lists what exists.
2. **Style with tokens, never hex.** Use `var(--primary)`, `var(--muted-foreground)` and the rest
   (`references/tokens.md`). A hardcoded colour breaks dark mode and the contrast audit.
3. **Label every control.** `Field` ties the label, hint and error to the control; never use a
   placeholder as the label.
4. **Name every icon-only control** with visible text or an `aria-label`.
5. **A dialog always has a title and a description.** `Dialog` requires both props.
6. **Values are text.** `Table` cells, toast messages and nav labels render as text. Never feed user
   input to `{@html}`.

## Pages

A page is `web/src/routes/<name>/+page.svelte` (see the `svelte-shell` skill for nav entries). Start
from `Showcase.svelte`, which is every piece on one page and what `kit.test.ts` audits.

| Need | Use |
|---|---|
| A form field | `<Field label hint error bind:value />`; `multiline` for a textarea; `type` for the input type |
| An action | `<Button variant="primary" \| "secondary" \| "destructive" \| "ghost">`; `type="submit"` in a form |
| Tabular data | `<Table caption columns rows />`; an empty `rows` shows an `EmptyState` |
| Nothing to show | `<EmptyState title description>` with an `action` snippet saying what to do next |
| A confirmation or small task | `<Dialog bind:open title description>` with an `actions` snippet |
| Feedback after an action | `toast.push({ message, tone })` (`tone`: `info`, `success`, `error`) |

`AppShell` already renders the one `<Toaster />`; do not render a second. Toasts never disappear on
a timer and each can be dismissed.

Patterns (`references/patterns.md`) cover forms with validation, destructive confirmation, empty
and error states, and navigation on a phone (bottom tab bar versus a menu-button drawer, overflow,
focus and ARIA).

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

Everything reads the custom properties in `tokens.css`, declared in the `albitor-kit` cascade layer so
any unlayered declaration wins. To theme the app, declare the same properties in
`web/src/lib/app/app.css`; do not edit `tokens.css` (it is managed and will be reset) or the
components. Dark mode follows the visitor's preference unless the page sets `data-theme="light"`,
and `data-theme="dark"` forces it.

If you override colours, keep `contrast.test.ts` passing. It computes WCAG 2.2 AA ratios (4.5:1 for
text, 3:1 for the control border, focus ring and primary fill) for every pair the components use,
in light and dark. `kit.test.ts` renders the shell pieces and runs axe on them; axe cannot judge
contrast without a layout engine, which is why the contrast test exists.

The general WCAG 2.2 AA content (criteria, audits, accessibility statements) lives in the
`test-playwright` plugin, not here.

## Accessibility built in

| Piece | What the kit already does |
|---|---|
| `AppShell` | Skip link first, one `main` landmark, a labelled `nav`, `aria-current="page"` on the current link, visible focus |
| `Field` | `label for`, `aria-describedby` to the hint and error, `aria-invalid`, an "Error:" prefix a screen reader reads |
| `Table` | A caption, `scope` on every header, a keyboard-focusable scroll region |
| `Dialog` | Named by its title, described by its description, focus trapped and returned (Bits UI) |
| `Toaster` | Always-present `status` (polite) and `alert` (assertive, for errors) regions |

## Design principles

These carry from the shadcn design system and hold for every screen, kit piece or not. The
detail, and the map from other shadcn components to their Bits UI primitive, is in
`references/design-system.md`.

- **The kit is the inventory.** A screen is built from kit components and Bits UI primitives. A
  hand-rolled dialog, menu, tabs or select is an ad-hoc component and a defect.
- **Semantic tokens only.** Roles (`--primary`, `--muted-foreground`), never a hex or a raw size.
  Every colour is a pair: text and its surface are chosen together.
- **Variants, not one-offs.** A new look is a new `variant` in the component, not a long style
  override at each call site.
- **Never remove the focus ring.** Replace an outline only with an equally visible indicator that
  keeps 3:1 against its neighbours.
- **Colour is never the only signal.** Pair a destructive or success colour with text or an icon.

## Do not

- Do not add Tailwind: this kit styles with scoped CSS over tokens, and the web-svelte build has no
  Tailwind plugin. If you do want it, that is a change to `vite.config.ts`, which is ejected
  first (see `svelte-shell`).
- Do not hand-roll focus management or portals; use Bits UI, which the dialog already wraps.
- Do not remove the focus outlines.
