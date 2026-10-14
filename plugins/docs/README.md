# Documentation frameworks plugin (`docs`)

A [Claude Code plugin](https://code.claude.com/docs/en/plugins) that packages three
authoritative documentation frameworks into skills, so role-agents produce an app's
documentation to a consistent, professional standard. These are Albitor's first-party
**`required-documentation`** capability skills (see [ADR-0053](https://github.com/albitor-ltd/albitor/blob/main/docs/decisions/0053-skill-selected-capabilities.md)).

Each framework's content is distilled from its authoritative source and bundled so the
assistant has the reference material to hand, with links back to the source for exact,
current detail.

## What's inside

### Skills (load automatically when relevant)

| Skill | Triggers on | Bundled references |
| --- | --- | --- |
| `docs:diataxis` | Planning, writing, or restructuring any documentation | The four modes — tutorials, how-to guides, reference, explanation |
| `docs:c4` | Creating or reviewing architecture diagrams | Notation rules, supplementary diagrams, worked Mermaid examples |
| `docs:arc42` | Writing an architecture / solution document | The 12 sections, section-by-section |

- **`diataxis`** — the [Diátaxis](https://diataxis.fr/) framework (Daniele Procida).
  The four documentation modes, the two axes that separate them, the compass for
  deciding which mode you're in, and the cardinal rule never to mix modes on a page.
- **`c4`** — the [C4 model](https://c4model.com/) (Simon Brown). The four hierarchical
  levels (System Context, Container, Component, Code), consistent notation, the
  supplementary diagrams (System Landscape, Dynamic, Deployment), which level to draw
  and which to skip, and tooling (Structurizr, Mermaid C4, PlantUML).
- **`arc42`** — the [arc42](https://arc42.org/) template (Starke & Hruschka). All 12
  architecture-documentation sections, what belongs in each, right-sizing them to
  avoid empty filler, and how arc42 composes with C4 (§3/§5/§6/§7) and ADRs (§9).

The three compose: **Diátaxis** structures the whole docs set, **arc42** structures the
architecture document within it, and **C4** supplies that document's diagrams.

## The `required-documentation` capability

Each skill's `SKILL.md` declares the capability via the agentskills.io-standard
`metadata` block:

```yaml
---
name: diataxis
description: <when to use>
metadata:
  capability: required-documentation
---
```

Albitor's governance scan reads `metadata.capability` at snapshot time and offers
these skills, grouped by capability, in the skill catalogue. An app that selects a
`required-documentation` skill has its containing plugin folded into the delivery set;
the build's done-criteria floor then presence-gates the required documentation
deliverables (ADR-0053 §4).

## Installing

Add the marketplace that lists this plugin, then install:

```bash
claude plugin marketplace add albitor-ltd/albitor-plugins
claude plugin install docs@albitor-plugins
```

Or load it directly for one session during development:

```bash
claude --plugin-dir /path/to/albitor-plugin-docs
```

Validate the plugin structure:

```bash
claude plugin validate /path/to/albitor-plugin-docs
```

## Keeping it current

The bundled references are a distilled snapshot, not a live mirror. The frameworks
evolve, so each skill tells the assistant to confirm exact, current detail against the
authoritative sources:

- Diátaxis — <https://diataxis.fr/>
- C4 model — <https://c4model.com/>
- arc42 — <https://arc42.org/>

To refresh, re-distil from those sources into `skills/*/references/` and bump the
`version` in `.claude-plugin/plugin.json`.

## Licensing

- **Plugin code** (manifest, skills wiring): MIT — see [`LICENCE`](LICENCE).
- **Bundled reference content** under `skills/*/references/` is an independent
  distillation of publicly-documented frameworks. Diátaxis, the C4 model, and arc42
  are the work of their respective authors (Daniele Procida; Simon Brown; Gernot Starke
  & Peter Hruschka); this plugin is unaffiliated and links to each authoritative source.
  arc42 itself is published under a Creative Commons licence.
