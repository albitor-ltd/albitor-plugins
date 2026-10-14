---
name: diataxis
description: Use when planning, writing, reviewing, or restructuring an app's documentation — a README, a docs set, a tutorial, a how-to, an API/config reference, or a conceptual explainer. Applies the Diátaxis framework by Daniele Procida — the four documentation modes (tutorials, how-to guides, reference, explanation), the two axes that separate them, and the cardinal rule never to mix modes on one page.
metadata:
  capability: required-documentation
---

# Diátaxis

Help authors produce documentation that serves users, by applying **Diátaxis** — a
systematic framework that recognises four distinct documentation modes and keeps
them apart. The core insight: documentation fails when it tries to be everything at
once. Split it by the *need it serves*, and both writing and reading get easier.

Authoritative source: <https://diataxis.fr/> (Daniele Procida). This skill is a
distilled, self-contained index — confirm current detail against the source.

## The one idea

There are **four** kinds of documentation, because users have four distinct needs.
Each need wants a different shape of content. Trying to satisfy two needs in one
document satisfies neither.

| Mode | Serves the need to… | Orientation | Analogy |
| --- | --- | --- | --- |
| **Tutorial** | *learn* by doing | learning / study | teaching a child to cook |
| **How-to guide** | *achieve a goal* | working / task | a recipe |
| **Reference** | *look something up* | working / information | an encyclopaedia entry |
| **Explanation** | *understand* | learning / study | an article on culinary history |

## The two axes (why there are exactly four)

Diátaxis places the four modes on a grid formed by two independent distinctions:

- **Action ↔ Cognition** — is the content about *practical steps* (doing) or about
  *theoretical knowledge* (thinking)?
- **Acquisition ↔ Application** — does it serve *study* (acquiring skill/knowledge,
  learning) or *work* (applying skill/knowledge to a task at hand)?

```
                 PRACTICAL (action)        THEORETICAL (cognition)
              ┌───────────────────────┬───────────────────────┐
  ACQUISITION │      TUTORIALS        │     EXPLANATION        │
  (study)     │  learning-oriented    │  understanding-oriented│
              ├───────────────────────┼───────────────────────┤
  APPLICATION │   HOW-TO GUIDES       │      REFERENCE         │
  (work)      │   goal-oriented       │  information-oriented  │
              └───────────────────────┴───────────────────────┘
```

Tutorials and how-tos share *action*; reference and explanation share *cognition*.
Tutorials and explanation share *study*; how-tos and reference share *work*. Every
mode is one corner — that is why there are four and no more.

## The cardinal rule: never mix modes

**One document does one job.** The most common documentation failure is a page that
starts as a tutorial, drifts into explaining *why*, then dumps a table of options,
then tells you how to accomplish a related task. Each of those belongs in a
different mode. When you catch yourself doing two jobs, split the page.

Signs you are mixing modes (fix them):

- A **tutorial** that pauses to explain design rationale → move the rationale to an
  explanation; link to it.
- A **how-to** that teaches concepts before the steps → assume competence; link to a
  tutorial/explanation for readers who need grounding.
- **Reference** that includes a walkthrough → strip it to facts; put the walkthrough
  in a how-to.
- **Explanation** with step-by-step instructions → move the steps to a how-to.

## The compass — deciding which mode you are writing

Ask two questions about the content in front of you:

1. Is it about **action** or **cognition**?
2. Does it serve **acquisition** (study) or **application** (work)?

| If it informs… | and serves… | then it is a… |
| --- | --- | --- |
| action | acquisition | **tutorial** |
| action | application | **how-to guide** |
| cognition | application | **reference** |
| cognition | acquisition | **explanation** |

If a paragraph answers "yes" to more than one cell, it is two documents.

## The four modes in one line each (deep detail in `references/`)

- **Tutorial** — a *lesson*. You take a beginner by the hand through a series of
  steps to a meaningful, successful result. It builds confidence and familiarity,
  not comprehensive knowledge. The author is responsible for the learner's success.
  → `references/tutorials.md`
- **How-to guide** — a *recipe*. It helps a competent user accomplish a specific
  real-world goal. It assumes knowledge, focuses on the goal, and omits teaching.
  → `references/how-to-guides.md`
- **Reference** — a *map*. Dry, accurate, complete technical description of the
  machinery (APIs, config keys, CLI flags, classes). Structured to mirror the code
  so it is fast to look things up. It describes; it does not instruct or explain.
  → `references/reference.md`
- **Explanation** — a *discussion*. It illuminates a topic from a higher vantage:
  why it works this way, the design decisions, alternatives, history, connections.
  Read away from the keyboard. → `references/explanation.md`

## How to structure a whole docs set

A complete documentation set usually has all four modes, kept in separate,
clearly-labelled sections — often the literal top-level folders `tutorials/`,
`how-to/`, `reference/`, `explanation/` (this is exactly how the Albitor `docs/`
tree is laid out). Guidance:

1. **Start where the user's need is sharpest.** A new project most urgently needs a
   *tutorial* (get someone to first success) and *reference* (so they can look
   things up). How-tos and explanation grow as questions recur.
2. **Let the set grow organically, one need at a time.** Don't pre-build empty
   scaffolding. When a support question keeps arriving, write the how-to. When
   people keep asking "but why", write the explanation.
3. **Cross-link across modes, don't merge them.** A how-to links to the reference
   for the flags it uses and to an explanation for the concepts behind it. Links are
   the seam that lets each page stay single-purpose.
4. **Name and place pages by mode.** "Getting started" (tutorial), "How to configure
   TLS" (how-to), "Configuration reference" (reference), "How authentication works"
   (explanation). The title should telegraph the mode.

## Working method (use this iteratively, don't boil the ocean)

Diátaxis is meant to be applied *incrementally*. You do not need a grand plan.

1. Look at the documentation (or the gap) in front of you **now**.
2. Use the compass to decide which single mode it should be.
3. Make it a better example of that mode; move out anything that belongs elsewhere.
4. Repeat. The overall structure emerges from many small, correct decisions.

## Common failure modes (and the fix)

- **The mega-README that does all four jobs** → split into a short tutorial-style
  quickstart, a reference section, and links out to how-tos/explanation.
- **Tutorials that aren't reliable** → a tutorial the learner can't complete
  verbatim is broken. Test every step end-to-end; pin versions; show expected
  output. The learner must succeed.
- **How-tos that teach instead of doing** → cut the concept-teaching; assume the
  reader knows what they're doing and wants the goal met.
- **Reference that editorialises** → strip opinion and instruction; state facts,
  structured to match the product.
- **"Explanation" that is really scattered notes** → give it a thesis and a topic
  boundary; it should read as a coherent discussion, not a grab-bag.
- **No explanation at all** → the product feels arbitrary. Explanation is what makes
  the rest make sense; don't skip it.

## When authoring docs for a delivered app

- Ship at minimum: a **tutorial** (get it running / first success) and a
  **reference** (API, config, CLI/env). Add **how-tos** for the top recurring tasks
  and **explanation** for the architecture and key decisions.
- Keep each page single-mode; link generously between them.
- Pair with the `c4` and `arc42` skills for the *architecture* documentation —
  those largely live in the **explanation** and **reference** quadrants.

## Related

- `c4` skill — architecture diagrams (the visual half of explanation/reference).
- `arc42` skill — the 12-section architecture-doc template; its narrative sections
  are Diátaxis *explanation*, and it embeds C4 diagrams.
