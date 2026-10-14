---
name: c4
description: Use when creating, reviewing, or discussing software architecture diagrams for an app — a System Context, Container, Component, or deployment diagram, or any "how the system fits together" picture. Applies the C4 model by Simon Brown — four hierarchical levels of abstraction, consistent notation, and which level to draw (and which to skip).
metadata:
  capability: required-documentation
---

# The C4 model

Help authors draw clear, consistent software architecture diagrams using **C4**
(Context, Containers, Components, Code) — a simple, hierarchical way to tell the
architecture story at multiple zoom levels, like Google Maps for code.

Authoritative source: <https://c4model.com/> (Simon Brown). This skill is a
distilled, self-contained index — confirm current detail against the source.

## The core idea: zoomable abstractions

C4 describes a software system with a small, fixed set of **abstractions**, then
lets you draw them at increasing levels of detail. You zoom in one level at a time,
and every diagram stands on its own.

### The abstraction hierarchy

A **person** uses a **software system**, which is made of **containers**, each made
of **components**, each made of **code** elements.

- **Person** — a human user/role/actor of the software (a persona, not an individual).
- **Software system** — the highest-level unit that delivers value; the thing you're
  building, plus the other systems it depends on. Often owned by one team.
- **Container** — a *separately runnable/deployable unit* that executes code or
  stores data: a web app, an API/service, a mobile app, an SPA, a database, a
  serverless function, a message queue, a file store. **NOT a Docker container** —
  the word predates Docker; think "something that hosts code or data". This is the
  most misunderstood C4 term.
- **Component** — a grouping of related functionality behind an interface, *inside* a
  container (e.g. a controller, a service, a repository). Not separately deployable.
- **Code** — classes, interfaces, functions — the implementation of a component.

## The four levels (Level 1 → 4)

You draw one diagram per level, zooming in each time:

| Level | Diagram | Zoom | Audience | Answers |
| --- | --- | --- | --- | --- |
| **1** | **System Context** | the system as one box + its users and the systems it talks to | everyone, incl. non-technical | "What is this system, who uses it, and what does it depend on?" |
| **2** | **Container** | inside the system box: the deployable/runnable units and how they talk | technical people (devs, ops) | "What are the high-level moving parts and their technologies?" |
| **3** | **Component** | inside one container: its components and their relationships | developers of that container | "How is this container structured internally?" |
| **4** | **Code** | inside one component: classes/functions (UML-ish) | rarely needed | "How is this component implemented?" |

- **Level 1 — System Context:** your system in the centre as a single box,
  surrounded by the people who use it and the external systems it integrates with.
  Deliberately hides all internal detail. Draw this for **every** system.
- **Level 2 — Container:** open the system box to show each app/service/data store as
  a box, labelled with its technology, and the connections (with protocols) between
  them and to external systems. This is usually **the most useful diagram** and the
  one most teams should always have.
- **Level 3 — Component:** open **one** container to show its major components and how
  they collaborate. Draw only for containers where the internal structure is worth
  explaining; not for every container.
- **Level 4 — Code:** open one component to class/function level. **Usually
  unnecessary** — it dates quickly and is better generated on demand by the IDE.
  Draw only for a genuinely complex, important component, and don't hand-maintain it.

### Which levels to actually draw

Start at the top and go only as deep as adds value:

- **Always:** System Context (1) and Container (2). These two carry most of the value.
- **Sometimes:** Component (3) — for the containers whose internals matter.
- **Rarely:** Code (4) — almost never worth hand-drawing; generate it if you need it.

Don't produce every level for every part of the system for the sake of completeness.

## Notation — keep it self-describing

C4 is **notation-independent** (no mandated shapes/colours), but a diagram must stand
alone. Enforce these rules — full detail in `references/notation.md`:

- **Every element** has a **name**, an element **type** (Person / System / Container /
  Component), and — for containers and components — a **technology** (e.g. "Go", "React
  SPA", "DynamoDB", "Lambda"), plus a short **description** of its responsibility.
- **Every relationship (line)** is directed and **labelled** with its intent
  ("Reads from", "Sends events to") and, ideally, technology/protocol ("JSON/HTTPS").
- **Every diagram** has a **title** (level + scope, e.g. "System Context diagram for
  Albitor") and a **key/legend** explaining shapes, colours, line styles, and
  acronyms. If a reader needs tribal knowledge to decode it, it's incomplete.
- Distinguish your **system's internals** from **external** systems/people (colour or
  a "[external]" marker).

## Supplementary diagrams (beyond the four levels)

C4 defines optional extras for when the four core diagrams aren't enough — detail and
examples in `references/notation.md`:

- **System Landscape** — a bigger-picture view of *multiple* software systems across
  an enterprise/domain and how they relate; zoomed out beyond a single System Context.
- **Dynamic** — shows a specific runtime *collaboration*: how a set of elements
  cooperate to satisfy one feature/use case, with numbered interaction steps (like a
  simplified UML communication/sequence diagram).
- **Deployment** — maps containers to *infrastructure*: environments, nodes,
  regions, clusters, serverless platforms. Shows *where the containers run*, not what
  they do.

## Tooling

C4 is **tooling-independent** — draw it however you like, but prefer
diagrams-as-code so it lives with the source and stays current:

- **Structurizr DSL** — the model-based, first-party approach: describe the model
  once, render every level/view from it. Best for a maintained architecture model.
- **Mermaid C4** — `C4Context` / `C4Container` / `C4Component` / `C4Dynamic` /
  `C4Deployment` blocks. Renders on GitHub, in Markdown, and in the Albitor Docs pane.
  **Use Mermaid for an Albitor app's in-repo architecture docs.**
- **PlantUML** with the C4-PlantUML macro library — mature, flexible, image output.

Prefer whichever renders inside the project's docs pipeline; keep the diagram source
next to the code so it's reviewed and updated with changes.

### Diagrams in the Albitor Docs pane

An app's `docs/` directory is read live and rendered in that app's **Docs** pane. Write
every diagram there as Mermaid.

- Write each diagram in a ```` ```mermaid ```` fence. The pane draws it.
- Do not draw a diagram as ASCII art in a fence. The pane renders that fence as a code
  block, and the drawing breaks on a reader's font or window width.
- Do not link an image file for a diagram. The pane renders the doc's own text, and a
  relative image path does not resolve.
- Put a Markdown heading above every diagram. The pane lists headings in a contents
  rail, so the heading is how a reader reaches the diagram.
- Do not set colours in a diagram. The pane redraws it in the reader's own light or
  dark theme, and hardcoded colours are then wrong in one of the two.

## When authoring an app's architecture docs

1. Draw the **System Context** first — orient every reader on what the system is and
   its boundaries.
2. Draw the **Container** diagram — the moving parts and their technologies. This is
   the one to keep evergreen.
3. Add **Component** diagrams only for containers whose internals need explaining.
4. Add a **Deployment** diagram to show where it runs (especially for serverless/AWS).
5. Skip **Code** unless a component is genuinely intricate.
6. Give every diagram a title and a key; label every box and every line.

## Related

- `arc42` skill — its §3 (Context) uses the System Context diagram, §5 (Building
  Block View) maps to Container/Component, §7 (Deployment) uses the Deployment diagram.
- `diataxis` skill — architecture diagrams live in the **explanation** and
  **reference** quadrants of a docs set.
