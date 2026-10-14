# GDS standards plugin (`gds`)

A [Claude Code plugin](https://code.claude.com/docs/en/plugins) that helps you build and audit **GOV.UK-compliant** services. It packages the **GOV.UK Design System** and **WCAG 2.2 AA accessibility** (plus the UK public sector accessibility regulations) into skills, commands, an audit agent, and an advisory hook.

It is one of the plugins Albitor makes available to its users. The standards content is distilled from the live GOV.UK and W3C sources and bundled so the assistant has authoritative reference material to hand (see [Licensing](#licensing)).

## What's inside

### Skills (load automatically when relevant)

| Skill | Triggers on | Bundled references |
| --- | --- | --- |
| `gds:govuk-design-system` | Building/reviewing a GOV.UK or public-sector frontend | All 34 components, page & "ask users for information" patterns, colour/typography/layout/spacing styles, and how to consume GOV.UK Frontend |
| `gds:govuk-accessibility` | Building/testing/advising on accessibility | All 50 WCAG 2.2 A/AA criteria, the PSBAR 2018 regulations, the accessibility-statement template, and how to test |

### Commands

| Command | Does |
| --- | --- |
| `/gds:component [name or need]` | Look up or scaffold a Design System component — macro **and** plain HTML, with accessibility notes. |
| `/gds:audit [target]` | WCAG 2.2 AA audit of a file, component, or the current changes; prioritised findings. |
| `/gds:accessibility-statement [details]` | Draft a GOV.UK-format accessibility statement from the model template. |
| `/gds:review [target]` | Review frontend changes against **both** the Design System and WCAG 2.2 AA. |
| `/gds:preview [open\|serve]` | Open or serve the crownless Design System preview page (see below). |

### Agent

- `accessibility-auditor` — a read-heavy subagent for a thorough WCAG 2.2 AA + Design System audit without polluting the main context. Used by `/gds:audit`, or invoke directly.

### Hook

- A non-blocking `PostToolUse` hook (`hooks/check-frontend.sh`) that runs after edits to frontend files (`.html`, `.njk`, `.svelte`, `.vue`, `.jsx`, `.tsx`, and more). It flags a **high-confidence** subset of issues — missing `alt`, missing `lang`, positive `tabindex`, click handlers on `<div>`/`<span>`, fake `href="#"` buttons — and surfaces them to Claude to fix. It never blocks and stays silent on clean files. It is a nudge, not a substitute for `/gds:audit`. Requires `jq`.

### Preview page

`preview/index.html` is a self-contained, static **kitchen-sink** showcase of the GOV.UK Design System — buttons, form fields (inputs, textarea, select, radios, checkboxes with conditional reveal, date input), error summary and inline errors, notification banners, tags, panel, tables, tabs, summary list, task list, accordion, breadcrumbs, phase banner, and the standard black header bar. It opens straight from `file://` with no build step (all CSS is inlined) and is served or opened by `/gds:preview`.

It is deliberately **crownless**: no crown logo, no "GOV.UK" logotype, no GDS Transport typeface, and no Open Government Licence / Crown-copyright statements — a neutral service name and a system font stack are used instead. This matches how the Design System is used for **non-government** services; shipping those assets off a `service.gov.uk` domain is a licensing breach (which this plugin's own auditor flags). Every Albitor design-system plugin ships a preview page as a standard capability, and this one has three consumers:

- **Create / describe-and-build flow** — a visual reference for the components and patterns available when scoping a build.
- **Post-build design review** — the baseline the delivered UI is reviewed against.
- **Handover pack** — a component showcase bundled with what's handed to the recipient.

### Preview tokens

`preview-tokens.json` is the small, generic token set Albitor's look preview paints this house style
from (albitor-ltd/albitor#3070): fonts, a four-step type scale, radius, density, elevation, and the
text, border, surface, background and accent colours. Albitor validates it against an allow-list at
ingest (hex colours, px/rem lengths, plain font-family names, fixed enums) and ignores a file that
fails. The values are GOV.UK Frontend: the text, secondary-text, border, light-grey and brand
colours, the 16/19/24/36px type scale (body-s, body, heading-m, heading-l) and square corners. The
font stack is GOV.UK Frontend's fallback (`arial, sans-serif`): this pack is crownless and ships no
GDS Transport.

## Installing

Add the marketplace that lists this plugin, then install:

```bash
claude plugin marketplace add albitor-ltd/albitor-plugins
claude plugin install gds@albitor-plugins
```

Or load it directly for one session during development:

```bash
claude --plugin-dir /path/to/albitor-plugin-gds
```

Validate the plugin structure:

```bash
claude plugin validate /path/to/albitor-plugin-gds
```

## Keeping it current

The bundled references are a distilled snapshot, not a live mirror. GOV.UK Frontend and the Design System change (the 2025 GOV.UK brand refresh, for example, changed several functional colour values), so the skills always tell the assistant to confirm exact, current detail against the live sources:

- GOV.UK Design System — <https://design-system.service.gov.uk/>
- GOV.UK Frontend — <https://frontend.design-system.service.gov.uk/>
- WCAG 2.2 — <https://www.w3.org/TR/WCAG22/>
- Accessibility regulations — <https://www.gov.uk/guidance/accessibility-requirements-for-public-sector-websites-and-apps>

To refresh the references, re-distil from those sources into `skills/*/references/` and bump the `version` in `.claude-plugin/plugin.json`.

## Licensing

- **Plugin code** (manifest, skills wiring, commands, agent, hook script): MIT — see [`LICENCE`](LICENCE).
- **Bundled reference content** under `skills/*/references/`: distilled from GOV.UK guidance and the GOV.UK Design System, which are © Crown Copyright and licensed under the [Open Government Licence v3.0](https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/). WCAG 2.2 is © W3C. See [`NOTICE`](NOTICE) for attribution.
