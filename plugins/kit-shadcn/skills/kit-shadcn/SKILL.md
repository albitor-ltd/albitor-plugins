---
name: kit-shadcn
description: How to build pages from the kit-shadcn component kit on a SvelteKit app composed from web-svelte (shadcn-style components on Bits UI, one token set). Use when a build adds or changes a page, form, table, dialog, toast or empty state under web/src, themes the app, or touches web/src/lib/kit.
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
and error states.

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

## Do not

- Do not add Tailwind: this kit styles with scoped CSS over tokens, and the web-svelte build has no
  Tailwind plugin. If you do want it, that is a change to `vite.config.ts`, which is ejected
  first (see `svelte-shell`).
- Do not hand-roll focus management or portals; use Bits UI, which the dialog already wraps.
- Do not remove the focus outlines.
