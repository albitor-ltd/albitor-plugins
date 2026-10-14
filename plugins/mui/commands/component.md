---
description: Look up or scaffold a Material component in the app's framework (Svelte SMUI/m3-svelte or React MUI), with props and accessibility notes.
argument-hint: "[component name or what you're trying to build]"
---

The user wants help with a Material Design component: **$ARGUMENTS**

Do the following:

1. Load the `mui-design-system` skill and read `references/components.md` and `references/frontend-conventions.md` (and `references/patterns.md` if the request describes a flow or screen rather than a single element).
2. **Detect the app's framework and Material library first** — Svelte/SvelteKit (SMUI or m3-svelte), or React/Next.js (`@mui/material`). Check `package.json` and the surrounding code. If it's a fresh SvelteKit app with no Material library yet, default to **SMUI** (see `frontend-conventions.md` for why) and note the alternative. **Do not switch the app's framework to fit a snippet.** If the framework is genuinely ambiguous, ask.
3. Identify the right component(s). If the request is vague (e.g. "let users pick one of a few options"), recommend the correct component and say why (e.g. a radio group for a few visible options, a select/autocomplete for many); mention any component they should *not* reach for (e.g. don't hand-roll a dropdown).
4. Provide a realistic snippet **in the detected framework**:
   - **Svelte (SMUI):** import from the component's package (`import Button, { Label } from '@smui/button'`, `import Textfield from '@smui/textfield'`); use Svelte syntax and note the variant mapping (SMUI `raised` ≈ MUI `contained`).
   - **Svelte (m3-svelte):** named imports from `m3-svelte` with Material 3 token styling.
   - **React (MUI):** per-path imports (`import Button from '@mui/material/Button'`) from `@mui/material`, `@mui/icons-material`, or an MUI X package where relevant.
5. Call out the **accessibility requirements** for that component (they hold across libraries): a text field needs a `label` (or `aria-label`); an icon-only button/Fab needs an `aria-label`; a dialog needs a labelled title (`aria-labelledby`/`aria-describedby`); status colour must be paired with text/icon (not colour alone); interactive controls must keep visible focus. Reference the relevant WCAG criterion.
6. Show how to style it the Material way for that library — **theme tokens**, not hardcoded hex or px, so it tracks the theme and dark mode (React MUI: `sx`/`styled()` with `theme.palette.*`; SMUI: theme Sass + `var(--mdc-theme-*)`; m3-svelte: `--md-sys-color-*`). See `references/styles.md`.
7. Match the surrounding codebase: detect the library version, TypeScript, and the framework meta-framework (SvelteKit / Next.js / Vite), and tailor the snippet to it.

If no argument was given, ask what they're building and list the component categories from the reference.
