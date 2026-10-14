Source: arc42 (https://arc42.org/), Dr. Gernot Starke & Dr. Peter Hruschka. Distilled reference — verify against the live source (arc42 is available under a Creative Commons licence).

# arc42 — section-by-section detail

The 12 sections, in their fixed order and numbering. For each: what belongs, what to
keep out, how much to write, and the C4 mapping where one applies.

---

## 1. Introduction and Goals

**Purpose:** orient the reader — what is being built and what matters most.

**Contents:**
- A short description of the requirements and driving forces (an extract/abstract, not
  a full requirements spec — link to that).
- The **top 3–5 quality goals** that most influence the architecture, prioritised for
  the major stakeholders (e.g. security, availability, time-to-market). These recur,
  made concrete, in §10.
- A **stakeholder table**: role/name, and what each expects from the architecture and
  its documentation.

**Keep out:** the full requirements document; more than five headline quality goals.
**Size:** small. Almost always worth completing.

---

## 2. Constraints

**Purpose:** record what limits design and implementation freedom, so later decisions
are understood in context.

**Contents:** technical constraints (mandated languages, frameworks, cloud provider,
standards to comply with), organisational constraints (team structure, budget,
schedule, ways of working), and conventions (coding standards, naming, tooling).
Constraints can be system-wide or organisation-wide.

**Keep out:** decisions you *made* freely (those are §4/§9) — constraints are things
you had no choice about. **Size:** small; a table is often enough.

---

## 3. Context and Scope

**Purpose:** delimit the system from its environment and name its external interfaces.

**Contents:**
- **Business context (always):** the system as one box, its external communication
  partners (users, neighbouring systems), and the domain-level inputs/outputs between
  them.
- **Technical context (optional):** the same boundary expressed in technical terms —
  channels, protocols, and how domain interfaces map to them.

**C4 mapping:** this is the **C4 System Context diagram** (Level 1). Add a table of
external interfaces if useful. **Size:** small–medium. Nearly always worth doing.

---

## 4. Solution Strategy

**Purpose:** summarise, briefly, the fundamental decisions that shape the whole
architecture — the "executive summary of how".

**Contents:** the key technology choices, the top-level decomposition approach, how
the most important quality goals (from §1) are to be achieved, and any relevant
organisational decisions. Each point short; the detail lives in later sections and in
§9 decisions.

**Keep out:** exhaustive justification (that's §9) and low-level detail (that's §5/§8).
**Size:** small — a page or less. High value for the effort.

---

## 5. Building Block View

**Purpose:** the **static structure** — how the system is decomposed into parts, and
those parts into sub-parts.

**Contents:** a hierarchy of **white boxes** (a container/module you open up to see
its contents) containing **black boxes** (parts described only by responsibility and
interface). Zoom in level by level:
- **Level 1:** the whole system as one white box → its top-level building blocks.
- **Level 2:** open an important building block → its internal blocks.
- **Level 3+:** continue only for blocks that warrant it.

For each black box, give its responsibility and interface; open it into a white box
only when the detail matters.

**C4 mapping:** Level 1 ≈ **C4 Container diagram**; deeper levels ≈ **C4 Component
diagrams**. **Size:** usually the **largest** section — but only go as deep as adds
value. Don't decompose to class level here (that's rarely needed; see C4 "Code").

---

## 6. Runtime View

**Purpose:** show **behaviour** — how building blocks collaborate at runtime in
important scenarios.

**Contents:** a handful of significant scenarios: primary use cases/features,
interactions at critical external interfaces, start-up/shutdown or admin operations,
and notable **error/exception** handling. Use numbered steps, sequence diagrams, or
activity diagrams.

**C4 mapping:** **C4 Dynamic diagrams** (or UML sequence diagrams). **Keep out:** every
possible path — pick the few scenarios that illuminate the architecture. **Size:**
medium; scenario count is a judgement call.

---

## 7. Deployment View

**Purpose:** show the **technical infrastructure** and how software maps onto it.

**Contents:** environments (dev/test/prod), the infrastructure elements (servers,
regions, availability zones, clusters, serverless platforms, devices, networks), the
topology, and the **mapping of building blocks (containers) to infrastructure nodes**.
Explain why the infrastructure is shaped this way if non-obvious.

**C4 mapping:** **C4 Deployment diagram**. Essential for cloud/serverless systems.
**Size:** medium.

---

## 8. Crosscutting Concepts

**Purpose:** capture the overarching rules, patterns, and solution approaches that
apply across many building blocks — so they're stated once, not repeated everywhere.

**Contents (as relevant, not all):** the domain/data model, chosen architecture
patterns and styles, persistence, session/state handling, security and authentication,
logging/observability, error handling, concurrency, internationalisation,
configuration, build/deploy conventions, testing approach. Group related concepts;
link building blocks (§5) and behaviour (§6) to the concept rather than restating it.

**Keep out:** duplication of what a single building block does. **Size:** medium–large;
depends how much genuinely crosscuts. High value for reducing repetition.

---

## 9. Architecture Decisions

**Purpose:** record the important, expensive, critical, large-scale, or risky
decisions — **with their rationale** — so future readers understand *why*, not just
*what*.

**Contents:** one entry per significant decision: context/problem, the decision taken,
the consequences, and the alternatives considered and rejected. This is the natural
home for **ADRs** (Architecture Decision Records) — keep each as its own short,
immutable-ish record and let this section index them.

**Keep out:** trivial or easily-reversed choices. **Size:** grows over the system's
life as decisions accumulate. Don't backfill everything — capture decisions as they're
made.

---

## 10. Quality Requirements

**Purpose:** make the quality goals concrete and **testable**.

**Contents:**
- A **quality tree** (e.g. an ISO 25010-style breakdown) giving a high-level overview
  of the qualities that matter and their priorities.
- **Quality scenarios:** each states a *stimulus* and the *measurable response* — e.g.
  "a returning user loads the dashboard: fully interactive within 2s at the 95th
  percentile", or "an unauthenticated request to a protected endpoint receives 401".
  Scenarios turn vague "-ilities" into things you can verify.

The top qualities here elaborate the goals named in §1.2. **Size:** medium.

---

## 11. Risks and Technical Debt

**Purpose:** surface, honestly, the known technical risks and accumulated debt.

**Contents:** identified risks (with likelihood/impact and any mitigation/plan) and
known technical debt — shortcuts taken, weak spots, things the team "feels miserable
about". Ordering by priority helps.

**Keep out:** blame; vague hand-waving. Be specific and actionable. **Size:** small —
but candour matters more than length.

---

## 12. Glossary

**Purpose:** a shared vocabulary so stakeholders mean the same thing by each term.

**Contents:** the important domain and technical **terms** with agreed definitions;
disambiguate overloaded words; include translations if the team works across
languages. **Size:** small; grows as needed.

---

## Right-sizing summary

- Sections are **placeholders, not a quota** — fill what informs, stub what doesn't.
- Match total effort to the system's **risk and lifespan**: a POC lives on §1/§3/§4 and
  a slim §5; a long-lived regulated system earns the full 12.
- **Never fabricate** content to fill a section. An honest "to be added" beats filler
  that trains readers to skip the document.
- Keep the **numbering and order fixed**; vary only the depth.
- Push shared ideas into **§8** and reference them; keep decisions in **§9**; put
  diagrams (C4) in **§3/§5/§6/§7** instead of prose.
- Keep it **beside the code and current** — reconcile the affected section with each
  change, the same way ADRs and reference docs travel with the code.
