# shadcn/ui design-system plugin (`shadcn`)

A [Claude Code plugin](https://code.claude.com/docs/en/plugins) that helps you build, audit, and preview modern web apps with **shadcn/ui** — accessible components on **Radix UI** primitives, styled with **Tailwind CSS**. It packages the design system and **WCAG 2.2 AA accessibility** into skills, commands, an audit agent, an advisory hook, and a standalone preview page.

**Archetype: the default modern SME / net-new web app.** shadcn/ui is the recommended design system for greenfield product builds — dashboards, CRUD apps, SaaS front-ends, internal tools. It gives a clean, contemporary look with a strong accessibility baseline (Radix) and no runtime lock-in (you own the copied-in component code). This plugin is one of the design-system plugins Albitor makes available to its users.

## What's inside

### Skills (load automatically when relevant)

| Skill | Triggers on | Bundled references |
| --- | --- | --- |
| `shadcn:shadcn-design-system` | Building/reviewing a modern React (or Svelte/Vue port) frontend | The shadcn/ui component registry + the Radix primitive each wraps, page & form patterns, the CSS-variable token/theming system, and how to consume shadcn/ui (CLI, `components.json`, `cn()`, `cva`) |
| `shadcn:shadcn-accessibility` | Building/testing/advising on accessibility | All 50 WCAG 2.2 A/AA criteria with shadcn/Radix-aware tips, how to test (axe/Playwright/jest-axe + manual + AT), and the accessibility-statement / VPAT template |

### Commands

| Command | Does |
| --- | --- |
| `/shadcn:component [name or need]` | Look up or scaffold a shadcn/ui component — install command, usage, the Radix primitive it wraps, accessibility notes. |
| `/shadcn:audit [target]` | WCAG 2.2 AA audit of a file, component, or the current changes; prioritised findings. |
| `/shadcn:review [target]` | Review frontend changes against **both** shadcn/Radix conformance (0 ad-hoc components) and WCAG 2.2 AA. |
| `/shadcn:accessibility-statement [details]` | Draft a voluntary accessibility statement / VPAT (WCAG 2.2 AA, EAA/ADA/508-aware). |
| `/shadcn:preview` | Open or serve the bundled component preview page (the kitchen-sink showcase). |

### Agent

- `shadcn-accessibility-auditor` — a read-heavy subagent for a thorough WCAG 2.2 AA + shadcn/Radix conformance audit without polluting the main context. Used by `/shadcn:audit`, or invoke directly. (Named with the `shadcn-` prefix so it doesn't collide with other design-system plugins.)

### Hook

- A non-blocking `PostToolUse` hook (`hooks/check-frontend.sh`) that runs after edits to frontend files (`.tsx`, `.jsx`, `.svelte`, `.vue`, `.astro`, `.html`, `.mdx`). It flags a **high-confidence** subset — missing `alt`, missing `lang`, positive `tabindex`, click handlers on `<div>`/`<span>`, and **ad-hoc components** (a raw `<button>`/`<input>`/`<select>`/`<textarea>` where a shadcn/Radix primitive exists, skipping the `components/ui/` definitions themselves). It never blocks and stays silent on clean files. It is a nudge, not a substitute for `/shadcn:audit`. Requires `jq`.

### Preview page

- `preview/index.html` — a **self-contained, build-free** kitchen-sink that renders the core shadcn/ui components (buttons and variants, inputs, form fields, card, dialog, tabs, table, badges, alert, checkbox, switch, select, tooltip, and more) in **light and dark** themes. It inlines all CSS/JS — no network, no build step, no server — so it opens straight from `file://`. It is the "what does shadcn/ui look like?" reference consumed by the create/describe-and-build flow, the design-review baseline, and the handover pack. Open it with `/shadcn:preview`.

### Preview tokens

`preview-tokens.json` is the small, generic token set Albitor's look preview paints this house style
from (albitor-ltd/albitor#3070): fonts, a four-step type scale, radius, density, elevation, and the
text, border, surface, background and accent colours. Albitor validates it against an allow-list at
ingest (hex colours, px/rem lengths, plain font-family names, fixed enums) and ignores a file that
fails. The values are the shadcn/ui default (zinc) theme: Tailwind's `font-sans` stack,
`text-sm`/`text-base`/`text-xl`/`text-3xl`, `--radius` 0.5rem, and `foreground`, `muted-foreground`,
`border`, `muted`, `background`, `primary` and `primary-foreground` as hex.

## Installing

Add the marketplace that lists this plugin, then install:

```bash
claude plugin marketplace add albitor-ltd/albitor-plugins
claude plugin install shadcn@albitor-plugins
```

Or load it directly for one session during development:

```bash
claude --plugin-dir /path/to/albitor-plugin-shadcn
```

Validate the plugin structure:

```bash
claude plugin validate /path/to/albitor-plugin-shadcn
```

## Keeping it current

The bundled references are a distilled snapshot, not a live mirror. shadcn/ui's registry grows and component APIs change, Radix and Tailwind release updates (shadcn/ui now supports Tailwind v4), so the skills always tell the assistant to confirm exact, current detail against the live sources:

- shadcn/ui — <https://ui.shadcn.com/docs>
- Radix UI Primitives — <https://www.radix-ui.com/primitives>
- Tailwind CSS — <https://tailwindcss.com/docs>
- WCAG 2.2 — <https://www.w3.org/TR/WCAG22/>

To refresh the references, re-distil from those sources into `skills/*/references/` and bump the `version` in `.claude-plugin/plugin.json`.

## Licensing

- **Plugin code** (manifest, skills wiring, commands, agent, hook script, preview page): MIT — see [`LICENCE`](LICENCE).
- **Bundled reference content** under `skills/*/references/`: distilled from the shadcn/ui, Radix UI, and Tailwind CSS docs (all MIT-licensed open source) and W3C WCAG 2.2 (© W3C, W3C Document Licence). See [`NOTICE`](NOTICE) for attribution.
