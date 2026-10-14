Source: Diátaxis (https://diataxis.fr/explanation/), Daniele Procida. Distilled reference — verify against the live source.

# Explanation (understanding-oriented)

Explanation is **discussion** that illuminates a topic from a higher vantage point.
It's the documentation you read *away from the keyboard* — to understand, to make
sense of the product, to see why it is the way it is. Think of a background article
or an essay, not instructions.

Position on the compass: **cognition + acquisition**. Theoretical (understanding) in
service of *study* (broadening knowledge, not doing a task right now).

## What a good explanation does

- **Provides context and background.** Why the thing exists, the problem it solves,
  how it came to be, how it fits with the rest.
- **Explains *why*: design decisions, trade-offs, and alternatives considered.** This
  is the home for rationale that would clutter a tutorial or how-to.
- **Discusses and weighs.** It can hold opinions, compare approaches, admit
  trade-offs, mention what was rejected and why. The other three modes can't do this.
- **Makes connections.** Links topics together, relates the product to wider ideas,
  patterns, and history — giving the reader a mental model.
- **Is bounded by a topic**, not by a task or a piece of machinery. "How authentication
  works", "Why we chose an event spine", "The concurrency model" — each is a coherent
  discussion with a thesis.

## What explanation must NOT do

- Don't give step-by-step instructions — that's a how-to or tutorial.
- Don't be a technical description of every parameter — that's reference.
- Don't try to instruct the reader to *do* anything; the reader is thinking, not
  working.
- Don't let it become a dumping ground of unstructured notes — it needs a thread.

## Why explanation is the mode most often missing

Because it isn't tied to an immediate task, explanation is the easiest to skip — and
its absence is why a product can feel arbitrary or opaque even when the reference is
complete. Explanation is the connective tissue that makes the other three modes make
sense. For an architecture-heavy app, explanation is where the "how it fits together"
narrative lives (and where the `c4` and `arc42` skills plug in).

## Relationship to ADRs

Architecture Decision Records are a specialised, per-decision form of explanation:
each records the context, the decision, and the consequences/alternatives for one
choice. arc42 §9 collects them. Broad explanation gives the reader the shape of the
system; ADRs give the point-by-point rationale. Both are explanation-mode.

## Writing checklist

- [ ] Anchor it to a **topic** and give it a thesis/through-line.
- [ ] Explain the *why* and the *how-it-fits*, not the *how-to-do*.
- [ ] Discuss trade-offs, alternatives, and history where they aid understanding.
- [ ] Make connections to related concepts and link across the docs set.
- [ ] Keep instructions and exhaustive parameter lists out — link to how-tos/reference.
- [ ] Write it to be read in an armchair, not followed at a terminal.
