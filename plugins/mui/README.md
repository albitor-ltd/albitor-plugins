# Material UI (MUI) plugin (`mui`)

A [Claude Code plugin](https://code.claude.com/docs/en/plugins) that helps you build, audit, and preview **Material Design** interfaces — implemented with the Material component library that fits the app's framework, the app keeping its own framework. It packages the shared Material system (colour, type, spacing, elevation, theming) and component inventory plus **WCAG 2.2 AA accessibility** into skills, commands, an audit agent, an advisory hook, and a self-contained preview page.

**Material Design is the design language; the library depends on the framework:** **Svelte / SvelteKit** (the default app template) → **SMUI** (Svelte Material UI) or **m3-svelte** (Material 3); **React / Next.js** → **@mui/material** (MUI); extensible to Vue/Angular/web components. The pack does **not** flip a Svelte app to React to "get MUI" — see `skills/mui-design-system/references/frontend-conventions.md`.

Material is the **default design system for consumer and general-purpose web apps**: broad, familiar, batteries-included, MIT-licensed, and free to theme to your own brand. This plugin is one of the design-system plugins Albitor makes available to its users; the reference content is distilled from the live MUI, SMUI, m3-svelte, and Material Design docs and bundled so the assistant has authoritative material to hand (see [Licensing](#licensing)).

## What's inside

### Skills (load automatically when relevant)

| Skill | Triggers on | Bundled references |
| --- | --- | --- |
| `mui:mui-design-system` | Building/reviewing a Material Design frontend (any framework) | The shared Material system (colour/type/spacing/elevation/shape/theming), the Material component inventory (inputs, data display, feedback, surfaces, navigation, layout, utils + MUI X data grid / pickers / charts), common app patterns, and how to consume Material per framework (Svelte SMUI/m3-svelte, React @mui/material) |
| `mui:mui-accessibility` | Building/testing/advising on accessibility | All 50 WCAG 2.2 A/AA criteria (with MUI-specific how-to-meet notes), how to test, and a consumer-product accessibility-statement template |

### Commands

| Command | Does |
| --- | --- |
| `/mui:component [name or need]` | Look up or scaffold a Material component in the app's framework (Svelte SMUI/m3-svelte or React MUI) — idiomatic snippet with theme tokens and accessibility notes. |
| `/mui:audit [target]` | WCAG 2.2 AA audit of a file, component, or the current changes; prioritised findings. |
| `/mui:review [target]` | Review frontend changes against **both** correct MUI usage and WCAG 2.2 AA. |
| `/mui:accessibility-statement [details]` | Draft an accessibility statement for a consumer/general-purpose product. |
| `/mui:preview [what]` | Open, explain, or derive from the self-contained MUI kitchen-sink preview page. |

### Agent

- `mui-accessibility-auditor` — a read-heavy subagent for a thorough WCAG 2.2 AA + MUI-conformance audit without polluting the main context. Used by `/mui:audit`, or invoke directly.

### Hook

- A non-blocking `PostToolUse` hook (`hooks/check-frontend.sh`) that runs after edits to frontend files (`.html`, `.jsx`, `.tsx`, `.svelte`, `.vue`, `.astro`, `.mdx`). It flags a **high-confidence** subset of issues — missing `alt`, missing `lang`, positive `tabIndex`, click handlers on `<div>`/`<span>`, fake `href="#"` buttons, plus (in JSX/TSX) hand-rolled `<button>`/`<input>`/`<select>`/`<textarea>` where an MUI component exists — and surfaces them to Claude to fix. It never blocks and stays silent on clean files. It is a nudge, not a substitute for `/mui:audit`. Requires `jq`.

### Preview page

- `preview/index.html` — a **self-contained, static** kitchen-sink showcasing the core MUI components (app bar, buttons, text fields, selects, checkboxes/radios/switches, chips, cards, tabs, a data table, alerts, a dialog, progress, and a list) on the Material colour/type/elevation/spacing scales. It has **no build step** and opens directly from `file://` in any browser (inline CSS, no external fonts/scripts/CDNs). It is consumed by the create / describe-and-build flow, the review baseline, and the handover pack.

### Preview tokens

`preview-tokens.json` is the small, generic token set Albitor's look preview paints this house style
from (albitor-ltd/albitor#3070): fonts, a four-step type scale, radius, density, elevation, and the
text, border, surface, background and accent colours. Albitor validates it against an allow-list at
ingest (hex colours, px/rem lengths, plain font-family names, fixed enums) and ignores a file that
fails. The values are the MUI default light theme: Roboto, `body2`/`body1`/`h6`/`h4`
(0.875/1/1.25/2.125rem), `shape.borderRadius` 4px, `primary.main` #1976d2, and the text and divider
colours as their opaque equivalents on white.

## Installing

Add the marketplace that lists this plugin, then install:

```bash
claude plugin marketplace add albitor-ltd/albitor-plugins
claude plugin install mui@albitor-plugins
```

Or load it directly for one session during development:

```bash
claude --plugin-dir /path/to/albitor-plugin-mui
```

Validate the plugin structure:

```bash
claude plugin validate /path/to/albitor-plugin-mui
```

## Keeping it current

The bundled references are a distilled snapshot, not a live mirror. Material and its libraries change (MUI's Grid/styling/theme API shifted across v4→v7; SMUI moved off the deprecated MDC-Web at v9; m3-svelte moves quickly), so the skills always tell the assistant to confirm exact, current detail against the live sources:

- Material UI (MUI, React) — <https://mui.com/>
- Svelte Material UI (SMUI) — <https://sveltematerialui.com/>
- m3-svelte (Material 3 for Svelte) — <https://github.com/KTibow/m3-svelte>
- Material Design — <https://m3.material.io/>
- WCAG 2.2 — <https://www.w3.org/TR/WCAG22/>

To refresh the references, re-distil from those sources into `skills/*/references/` and bump the `version` in `.claude-plugin/plugin.json`.

## Licensing

- **Plugin code** (manifest, skills wiring, commands, agent, hook script, preview page): MIT — see [`LICENCE`](LICENCE).
- **Bundled reference content** under `skills/*/references/`: a distilled summary of the MUI documentation (MIT-licensed), Material Design (© Google), and WCAG 2.2 (© W3C, used under the W3C Document Licence). See [`NOTICE`](NOTICE) for attribution. Material UI, MUI, Material Design and Roboto are trademarks of their respective owners; this plugin is an independent aid and is not affiliated with or endorsed by MUI or Google.
