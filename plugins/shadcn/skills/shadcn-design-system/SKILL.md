---
name: shadcn-design-system
description: Use when building, reviewing, or discussing a modern web app frontend that should follow shadcn/ui — any React/Next.js (or Svelte/Vue port) UI built on shadcn/ui components, Radix UI primitives, and Tailwind CSS. Covers the component registry and the Radix primitive each wraps, page and form patterns, the CSS-variable theming / Tailwind token system, and how to consume shadcn/ui (copy-in ownership model, the CLI, cn(), cva variants) with accessibility built in.
metadata:
  capability: design-system
---

# shadcn/ui design system

Help developers build interfaces with **shadcn/ui** so they look and behave like a modern, accessible web app — the default choice for net-new SME, startup, and internal web apps.

## What shadcn/ui actually is (this changes how you use it)

shadcn/ui is **not** a component library you install as a runtime dependency. It is a collection of **accessible, composable components you copy into your own project** and own outright. The CLI (`npx shadcn@latest add button`) writes the component's source into `components/ui/` (or your configured path); from then on it is *your* code — you edit it directly, and there is no upstream package to upgrade.

Its three foundations:

1. **Radix UI primitives** — unstyled, fully accessible behaviour (focus management, keyboard interaction, ARIA roles/states, portalling, typeahead). Most interactive shadcn components (Dialog, Dropdown Menu, Select, Tabs, Tooltip, Popover, Accordion, Switch, Checkbox, Radio Group, Slider, …) are thin styled wrappers over a Radix primitive. This is where the accessibility baseline comes from.
2. **Tailwind CSS** — utility classes for all styling, with design tokens exposed as CSS variables so theming and dark mode are a variable swap, not a rewrite.
3. **A small set of conventions** — `cva` (class-variance-authority) for variants, a `cn()` helper (clsx + tailwind-merge) for conflict-safe class merging, and `components.json` describing the project's setup.

## Core principles (apply these first)

1. **Use a shadcn/Radix component if one exists.** There is a vetted, accessible component for almost every common need — dialogs, dropdowns, selects, tabs, tooltips, forms, tables, toasts. Reach for it before hand-rolling markup. Re-implementing a `<div>` modal or a bare `<button>` throws away the Radix accessibility baseline. The full registry, and the Radix primitive each wraps, is in `references/components.md`. **This is the "0 ad-hoc components" bar: build from the primitives, don't re-implement them.**
2. **Style with tokens, not hardcoded values.** Use the semantic Tailwind tokens (`bg-primary`, `text-muted-foreground`, `border-input`, `ring-ring`, `bg-destructive`) that map to CSS variables. Don't hardcode hex or scatter arbitrary values (`bg-[#0f172a]`) — that breaks theming, dark mode, and consistency. See `references/styles.md`.
3. **Express variants through the component, not class soup.** shadcn components use `cva` to define `variant`/`size` props (e.g. `<Button variant="destructive" size="sm">`). Extend those variants rather than piling on one-off classes; merge overrides with `cn()` so tailwind-merge resolves conflicts predictably.
4. **Compose with patterns, not just components.** A settings form, a data table with row actions, a command palette, a confirm-destructive-action flow are *patterns* — tested arrangements of components. See `references/patterns.md`.
5. **Accessibility is not optional, and Radix is a baseline not a guarantee.** Radix gives correct keyboard/focus/ARIA behaviour for free, but you still own labels, contrast, alt text, heading order, and accessible names for icon-only controls. Pair this with the `shadcn-accessibility` skill when auditing.

## Bundled references — read the relevant one before answering in detail

| File | When to read it |
| --- | --- |
| `references/components.md` | Choosing/implementing any UI element. Lists the shadcn/ui registry, the Radix primitive (or other dep) each wraps, whether it is interactive, when (not) to use it, and accessibility gotchas. This is the conformance inventory. |
| `references/patterns.md` | Designing a page or a flow — forms, settings pages, data tables, dialogs/confirmations, command palettes, empty states, auth screens, dashboards. |
| `references/styles.md` | The CSS-variable token system, colour/theming, dark mode, `cva` variants, `cn()`, radius/typography/spacing, focus rings, the `new-york` vs `default` style. |
| `references/frontend-conventions.md` | Installing/using shadcn/ui — the CLI, `components.json`, Tailwind v3 vs v4 setup, RSC/Client Components, and porting to Svelte/Vue. |

These are distilled from the live shadcn/ui, Radix UI, and Tailwind CSS docs. For exact, current detail (component APIs change, and the registry grows) always confirm against <https://ui.shadcn.com/docs>, <https://www.radix-ui.com/primitives>, and <https://tailwindcss.com/docs>.

## Working rules

- **Don't hardcode colours.** Use the semantic token classes backed by CSS variables so the app tracks the theme and dark mode. Define new colours as variables in `globals.css`, not inline. See `references/styles.md`.
- **Own the components, keep them consistent.** Because the code is copied in, you *can* edit `components/ui/*` — but keep edits principled (extend `cva` variants, keep the Radix wiring intact) so the set stays coherent and re-runnable. Don't strip Radix props, `data-state`, or focus-visible rings.
- **Forms:** every control needs an associated `Label` (`htmlFor` ↔ `id`), or use the shadcn `Form` component (React Hook Form + Zod) which wires `FormLabel`/`FormControl`/`FormMessage`/`aria-describedby` and error announcement for you. Never use placeholder text as the label. Validate on the server too.
- **Icon-only controls need an accessible name** — `aria-label` or visually-hidden (`sr-only`) text. A `<Button size="icon">` with only an SVG inside is invisible to screen readers otherwise.
- **Dialogs/Sheets need a title** — always render `DialogTitle` (visually hidden with `sr-only` if the design has no visible title) so the dialog has an accessible name; add `DialogDescription` where useful.
- **Client interactivity:** interactive Radix-based components are Client Components — in the Next.js App Router they need `"use client"`. Let Radix own the keyboard/focus behaviour; don't re-implement it.
- When starting a new app, initialise with `npx shadcn@latest init` (pick style, base colour, and CSS-variables theming), then add only the components you use.

## Related

- `shadcn-accessibility` skill — WCAG 2.2 AA criteria and how to test; the standard these components help you meet.
- Commands: `/shadcn:component`, `/shadcn:review`, `/shadcn:audit`, `/shadcn:preview`.
