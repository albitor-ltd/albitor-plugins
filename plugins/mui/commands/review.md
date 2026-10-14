---
description: Review frontend changes against both Material UI (MUI) usage and WCAG 2.2 AA.
argument-hint: "[file/dir path, or leave blank for the current changes]"
---

Review the MUI frontend in scope: **$ARGUMENTS** (if empty, review the current uncommitted changes; fall back to recently edited frontend files).

Load **both** skills: `mui-design-system` and `mui-accessibility`. Then review on two axes:

1. **Material Design conformance** (`references/components.md`, `patterns.md`, `styles.md`, `frontend-conventions.md`) — first detect the framework and its Material library (Svelte SMUI/m3-svelte, or React `@mui/material`) and review against that library's idiom:
   - Are vetted Material components used where one exists, instead of bespoke markup (`<button>`/`<input>`/`<select>` that should be the library's Button / TextField / Select)?
   - Are components used idiomatically (correct props/variants, options inside a select, a real button/link for interactive list rows) rather than fought against?
   - Is styling done with theme tokens (MUI React: `sx` + `theme.palette`/`theme.spacing`/`theme.typography`; SMUI: theme Sass + `var(--mdc-theme-*)`; m3-svelte: `--md-sys-color-*`) instead of hardcoded hex/px, so light/dark mode and the scale hold?
   - Is the theme + baseline established once (React MUI: a single `ThemeProvider` + `CssBaseline` and one styling engine; SMUI: the compiled Sass theme; m3-svelte: the M3 token stylesheet)?
   - Is the app kept in its own framework rather than partly rewritten to fit the design system?

2. **Accessibility** (`references/wcag-2.2.md`): run the WCAG 2.2 AA checks — alt text and icon-button `aria-label`, form labels/helper text, focus visibility, heading order, contrast, `lang`, announced + linked errors, target size, keyboard operability, dialog labelling.

Report findings grouped by axis, each with `file:line`, severity (Blocker/Major/Minor), the problem, and a concrete fix. Note any issue that needs a real-browser or assistive-tech check rather than static review. Finish with a prioritised to-do list. Be specific and code-level; don't restate the standards in the abstract.
