---
name: arc42
description: Use when writing or reviewing an app's architecture documentation — an architecture overview, a solution/design document, or a structured record of how and why a system is built the way it is. Applies the arc42 template by Starke and Hruschka — the 12 standard sections, what belongs in each, right-sizing them, and how arc42 composes with C4 diagrams and ADRs.
metadata:
  capability: required-documentation
---

# arc42

Help authors document a software architecture with **arc42** — a proven, 12-section
template that gives architecture documentation a predictable home for every kind of
information, so nothing important is forgotten and readers always know where to look.

Authoritative source: <https://arc42.org/> (Dr. Gernot Starke & Dr. Peter Hruschka).
This skill is a distilled, self-contained index — confirm detail against the source.

## What arc42 is (and isn't)

arc42 is a **template**: a set of 12 numbered sections, each with a clear purpose.
It is *not* a process, a method, or a mandate to fill every section to the brim. You
put "what" and "why" your architecture is the way it is into a stable structure that
stakeholders learn once and reuse across systems. It's tool-agnostic (Markdown, wiki,
AsciiDoc, Word) and pairs naturally with C4 for diagrams and ADRs for decisions.

## The 12 sections at a glance

| # | Section | Answers | Typical size |
| --- | --- | --- | --- |
| 1 | Introduction and Goals | What are we building and what matters most? | small |
| 2 | Constraints | What limits our freedom? | small |
| 3 | Context and Scope | Where are the system's boundaries? | small–medium |
| 4 | Solution Strategy | What are the big, shaping decisions? | small |
| 5 | Building Block View | How is it decomposed (static structure)? | **large** |
| 6 | Runtime View | How does it behave (key scenarios)? | medium |
| 7 | Deployment View | Where does it run (infrastructure)? | medium |
| 8 | Crosscutting Concepts | What rules/patterns apply everywhere? | medium–large |
| 9 | Architecture Decisions | Which choices, and why? | grows over time |
| 10 | Quality Requirements | What quality goals, made concrete? | medium |
| 11 | Risks and Technical Debt | What are we worried about? | small |
| 12 | Glossary | What do our terms mean? | small |

Full section-by-section detail (what belongs in each, what to leave out, and the C4
mapping) is in `references/sections.md`.

## The sections in brief

1. **Introduction and Goals** — a short statement of requirements and driving forces,
   the **top 3–5 quality goals** (the ones that most shape the architecture), and a
   table of the important **stakeholders** and what each expects from the docs.
2. **Constraints** — anything that limits design/implementation freedom: technical
   (mandated language, cloud, standards), organisational (team, budget, timeline),
   and conventions. Things you *must* work within, not things you chose.
3. **Context and Scope** — the system's boundary and its external partners
   (users, neighbouring systems) with the interfaces between them. A **business**
   context (always) and optionally a **technical** context. *Use a C4 System Context
   diagram here.*
4. **Solution Strategy** — a short summary of the fundamental, architecture-shaping
   decisions: key technologies, top-level decomposition approach, how the top quality
   goals are achieved, and relevant organisational choices. The executive summary of
   the "how".
5. **Building Block View** — the **static** decomposition as a hierarchy of "white
   boxes" containing "black boxes", zoomed in level by level (Level 1: whole system;
   Level 2: inside a block; etc.). *This is C4's Container/Component views.* Usually
   the largest section.
6. **Runtime View** — how the building blocks **behave** in a handful of important
   scenarios/use cases: key flows, interactions at critical interfaces, and
   error/exception behaviour. *Use C4 Dynamic diagrams or sequence diagrams.* Pick a
   few illustrative scenarios; don't document every path.
7. **Deployment View** — the **technical infrastructure** (environments, nodes,
   regions, topology) and the **mapping of building blocks to infrastructure**. *Use a
   C4 Deployment diagram here.*
8. **Crosscutting Concepts** — principal regulations and solution approaches that
   apply across many building blocks: domain model, chosen patterns/styles,
   persistence, security, logging/observability, error handling, i18n, etc. Concept
   once, referenced everywhere — keeps sections 5–7 from repeating themselves.
9. **Architecture Decisions** — the important, expensive, risky, or hard-to-reverse
   decisions **with their rationale**. This is the natural home for **ADRs** (one
   record per decision: context, decision, consequences, alternatives).
10. **Quality Requirements** — the quality goals from §1 made concrete and testable,
    as a **quality tree** plus **scenarios** (given a stimulus, the measurable
    response). This is how "must be secure/fast/maintainable" becomes verifiable.
11. **Risks and Technical Debt** — known technical risks and accumulated debt, ideally
    with mitigation. The honest "what worries the team" section.
12. **Glossary** — the domain and technical **terms** with agreed definitions, so
    stakeholders share one vocabulary (plus translations if multilingual).

## Right-sizing: the most important discipline

arc42's sections are **placeholders, not a quota**. The biggest failure mode is a
document padded with empty or boilerplate sections that readers learn to ignore.

- **Write only what carries information.** If a section has nothing meaningful yet,
  leave a one-line "not yet relevant / to be added" placeholder rather than inventing
  filler. An honest gap beats fabricated content.
- **Match effort to risk and lifespan.** A throwaway POC needs §1, §3, §4, and a
  slim §5 — little more. A long-lived, regulated system justifies the full set.
- **Don't repeat yourself across sections.** Put a concept in §8 once and reference it;
  don't restate it in every building block.
- **Prefer diagrams and tables to prose.** A labelled C4 diagram in §3/§5/§7 says more
  than paragraphs.
- **Keep it close to the code and current.** Documentation that drifts from reality is
  worse than none. Update the affected section with the change.
- **Order is fixed, depth is yours.** Keep the 12 sections in their standard order and
  numbering (readers rely on it), but let each be as short or deep as it warrants.

## How arc42 composes with C4 and ADRs

arc42 says *what to document*; C4 and ADRs supply *how* for the visual and
decision parts:

- **C4 diagrams** slot into arc42: **§3** ← System Context; **§5** ← Container +
  Component (Building Block View); **§6** ← Dynamic; **§7** ← Deployment. Use the `c4`
  skill for the diagrams and this template for the surrounding narrative.
- **ADRs** are the content of **§9**. Keep each decision as its own record; §9 indexes
  them. (Albitor keeps ADRs in `docs/decisions/`.)
- The narrative sections are Diátaxis **explanation** (see the `diataxis` skill); the
  interface/quality catalogues lean **reference**.

## When authoring an app's architecture doc

1. Always fill **§1 (goals + quality goals + stakeholders)**, **§3 (context, with a C4
   System Context diagram)**, and **§4 (solution strategy)** — these orient every reader
   cheaply.
2. Add **§5 (building blocks, C4 Container/Component)** and **§7 (deployment, C4
   Deployment)** for anything beyond a toy.
3. Use **§6** for the two or three flows that aren't obvious, **§8** for the
   crosscutting rules, and **§9** to point at the ADRs.
4. Keep **§10–§12** proportional; a short quality-scenario table, an honest risk list,
   and a real glossary add a lot for little effort.
5. Don't pad. Placeholder-and-move-on beats filler.

## How the Albitor Docs pane reads an app's `docs/`

An app's `docs/` directory is read live from the default branch and rendered in that
app's **Docs** pane. The pane is not a file browser, so write to these rules.

**Navigation.** The pane lists one entry per file, in a sidebar.

- Title each file with a first-level Markdown heading, or a front-matter `title`. The
  pane uses it as the sidebar entry. Without either it falls back to the filename.
- Put related files in one directory. The pane groups by the first directory under
  `docs/`, or by a front-matter `group`. Keep ADRs in `docs/decisions/`.
- Name a file so it sorts where a reader expects. The pane sorts by title inside a
  group. Zero-pad an ADR number (`0007-`, not `7-`).
- Write `docs/README.md` as an index. Files directly under `docs/` are listed before
  every group, so the index lands first.

**Links between documents.** The pane resolves a relative link against the directory of
the document that carries it.

- Link to another document by its relative path, including the `.md` extension:
  `[ADR-0007](decisions/0007-last-write-wins.md)`. The pane opens it.
- Do not link to a directory. `[the decisions](decisions/)` reaches no document. The
  pane strips such a link and leaves its text as prose.
- Do not assume a repository URL. Write the relative path, not a GitHub link.

**Sections.** The pane gives every heading an anchor and lists `##` and `###` headings
in a contents rail beside the document.

- Break a long document into `##` sections. The rail is how a reader navigates it, and
  a section anchor is what a reader sends a colleague.
- Keep each heading distinct within a document. Two identical headings both work, but
  the rail then shows the same label twice.

**Anything else.** The pane renders Markdown, plus Mermaid diagrams (see the `c4`
skill). It does not run scripts, load images, or resolve links out of `docs/`.

## Related

- `c4` skill — the diagrams that populate §3, §5, §6, §7.
- `diataxis` skill — arc42 narrative is explanation-mode; place it in the docs set
  accordingly.
