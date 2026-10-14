---
name: mui-design-system
description: Use when building, reviewing, or discussing a Material Design frontend — implemented with the framework-appropriate Material component library (Svelte/SvelteKit → SMUI or m3-svelte; React → @mui/material; extensible to Vue/Angular/web components), the app keeping its own framework. Covers the shared Material system (colour, typography, spacing, elevation, shape, theming) + accessibility, the Material component inventory (inputs, data display, feedback, surfaces, navigation, layout, plus MUI X data grid / date pickers / charts), common app patterns, and how to consume Material per framework with sensible defaults.
metadata:
  capability: design-system
---

# Material Design (MUI / Material UI)

Help developers build interfaces that follow Google's **Material Design** — so screens look consistent, are accessible by default, and are quick to assemble from vetted components. Material is the default design language for **consumer and general-purpose web apps**: broad, familiar, and batteries-included.

**Material Design is the design language; the component library depends on the app's framework — and the app keeps its framework.** Do not flip a Svelte app to React (or hand-approximate Material with no component library) to "get MUI". Use the Material library that fits the framework:

- **Svelte / SvelteKit** (the default app template) → **SMUI** (Svelte Material UI, default) or **m3-svelte** (when Material 3 fidelity is wanted). **Do this path first — it is the default template.**
- **React / Next.js** → **@mui/material** (MUI), Material's original and broadest implementation.
- **Vue / Angular / web components** → Vuetify / Angular Material / Material Web.

See `references/frontend-conventions.md` for which library to pick per framework and how to install/theme/use each (Svelte first). These libraries are MIT-licensed and free to use, ship, and modify. There is no branding restriction: **theme it to your own brand** — the point of Material is a consistent, accessible component system, not a fixed visual identity.

## Core principles (apply these first)

1. **Use a Material component if one exists.** Every Material library (SMUI, m3-svelte, @mui/material, …) has a vetted, accessible component for almost every common need — buttons, text fields, selects, dialogs, tables, menus, tabs, and more. Reach for it before hand-rolling `<button>`/`<input>`/`<div onClick>`. The full inventory is in `references/components.md` — this is also the conformance inventory for a "no ad-hoc components" check (shown in the `@mui`/React form; the same components exist in the Svelte libraries under their own import paths).
2. **Theme, don't hardcode.** Set your palette, typography, shape and spacing once (a `createTheme` in MUI React; a Sass theme in SMUI; M3 design tokens in m3-svelte) and consume tokens everywhere. Never scatter hex colours or magic pixel values — that breaks dark mode and the type/spacing scale. See `references/styles.md` for the system and `references/frontend-conventions.md` for the per-framework theming API.
3. **One theme, one baseline.** Establish the theme and a normalised baseline once (React MUI: a single `ThemeProvider` + `<CssBaseline />`; SMUI: the compiled Sass theme; m3-svelte: the M3 token stylesheet). See `references/frontend-conventions.md`.
4. **Compose with patterns, not just components.** A form, a dialog flow, an app-bar-plus-drawer shell, a data table with toolbar, a settings page — these are *patterns* built from components. See `references/patterns.md`.
5. **Accessibility is not optional.** Material components are accessible out of the box, but only if used correctly: label your text fields, give icon-only buttons an `aria-label`, label your dialogs, and never convey status by colour alone. Pair this with the `mui-accessibility` skill when auditing.

## Bundled references — read the relevant one before answering in detail

| File | When to read it |
| --- | --- |
| `references/components.md` | Choosing/implementing any UI element. Enumerates the Material component inventory by category (inputs, data display, feedback, surfaces, navigation, layout, utils) plus the MUI X packages (Data Grid, Date/Time Pickers, Charts, Tree View), with each component's purpose, when (not) to use it, and its accessibility requirements. Shown in the `@mui`/React form; the same components exist in the Svelte libraries. |
| `references/patterns.md` | Designing a screen or a flow — forms, dialogs, app shell/navigation, data tables, feedback, responsive layout, auth screens, settings, empty/loading/error states. Framework-agnostic arrangements of Material components. |
| `references/styles.md` | The shared, framework-agnostic Material system: colour palette & modes (light/dark), typography scale, spacing (8px unit), elevation/shadows, shape/radius, breakpoints, and the theme object. |
| `references/frontend-conventions.md` | **The per-framework consume layer** — which Material library to use for your framework (Svelte → SMUI/m3-svelte first; React → @mui/material; others), and how to install, theme, and use each. |

These are distilled from the live MUI documentation (<https://mui.com/>), the Svelte Material libraries (SMUI <https://sveltematerialui.com/>, m3-svelte <https://github.com/KTibow/m3-svelte>), and Material Design (<https://m3.material.io/>). They are a fast index — for exact, current props and behaviour always confirm against the library's own API pages, especially across major versions (MUI has evolved its Grid/styling/theme API across v4–v7; SMUI moved off MDC-Web at v9; m3-svelte moves quickly).

## Working rules

- **Match the framework first.** Detect the app's framework and use its Material library (Svelte → SMUI/m3-svelte; React → @mui/material) — never rewrite the app's framework to fit the design system. See `references/frontend-conventions.md`.
- **Don't hardcode hex colours or pixels.** Set brand palette, spacing, and type once in the theme and reference tokens (MUI React: `theme.palette.primary.main`, `theme.spacing(2)`; SMUI: theme Sass + `var(--mdc-theme-*)`; m3-svelte: `--md-sys-color-*`) so the UI tracks the theme and dark mode. See `references/styles.md`.
- **Forms:** every text field needs a `label` (or `aria-label`); use helper text for hints and the error state + helper text for validation. Group radios with a group label. Don't use placeholder text as the only label.
- **Icon-only controls:** any icon-only button/Fab **must** have an `aria-label` — otherwise it has no accessible name.
- **Dark mode & branding:** switch modes through the theme and brand via the palette — don't fork components or override with `!important`. Keep components unmodified in behaviour; restyle through the theme.
- When starting a new app, set up the theme, the baseline, and a light/dark toggle first (per your library — see `references/frontend-conventions.md`), then build screens from components.

## Related

- `mui-accessibility` skill — WCAG 2.2 AA criteria, how to test, and the accessibility-statement template.
- Commands: `/mui:component`, `/mui:review`, `/mui:audit`, `/mui:preview`.
- `preview/index.html` — a self-contained static kitchen-sink showcasing the core components.
