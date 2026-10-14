---
description: Open, explain, or adapt the MUI kitchen-sink preview page (a self-contained static showcase).
argument-hint: "[what you want — 'open', 'explain', or a component/screen to derive from it]"
---

The user wants to work with the Material UI (MUI) preview page: **$ARGUMENTS**

The plugin ships a **self-contained, static** kitchen-sink preview at `${CLAUDE_PLUGIN_ROOT}/preview/index.html`. It renders Material-styled markup with inline CSS and **no build step** — it opens straight from `file://` in any browser. It exists so the create / describe-and-build flow, the review baseline, and the handover pack all have a visual reference for how MUI components should look and behave.

Do the following, depending on what was asked:

1. **Open / show it:** point the user at `preview/index.html` and tell them it opens directly in a browser (no server, no `npm`). Summarise what it demonstrates: the app bar, buttons (contained / outlined / text), text fields and selects, checkboxes / radios / switches, chips, cards, tabs, a data table, alerts, a dialog, progress indicators, and a list — laid out on the Material colour, type, elevation and spacing scales, in light and dark.
2. **Explain a component:** read the relevant section of the preview markup and relate it to the real MUI component (props/variant) via `mui-design-system/references/components.md` and `styles.md`. The preview is a static approximation — the real component is the source of truth; note where behaviour (ripple, focus ring, menu opening) only exists in the live component.
3. **Derive a screen/component:** use the preview as the visual target and produce real code **in the app's framework** — Svelte (SMUI/m3-svelte) or React (`@mui/material`); detect it, don't assume React — with theme tokens, wiring up the accessibility requirements (labels, `aria-label` on icon buttons, dialog labelling). Load `mui-design-system` and see `references/frontend-conventions.md` for the per-framework idiom.
4. **Keep it self-contained:** if you change the preview, it must stay a single standalone HTML file that opens from `file://` — inline all CSS, no external fonts/scripts/CDNs, no build step. Verify it still opens standalone after any edit.

If no argument was given, do (1): tell the user where the preview is and what it shows.
