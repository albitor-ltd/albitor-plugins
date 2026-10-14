Source: Diátaxis (https://diataxis.fr/reference/), Daniele Procida. Distilled reference — verify against the live source.

# Reference (information-oriented)

Reference material is the **technical description** of the machinery and how to
operate it — the dry, accurate, complete facts. Think encyclopaedia entry or map: a
user consults it while working to look something up, not to learn or to be persuaded.

Position on the compass: **cognition + application**. Theoretical (it's information,
not action) in service of *work* (the reader has a task and needs a fact).

## What a good reference does

- **Describes, and only describes.** State what things are, what they do, their
  parameters, types, defaults, return values, error conditions. No teaching, no
  opinion, no goal-directed instruction.
- **Is accurate and complete.** Reference is the authority. If it's wrong or has
  gaps, the reader is stranded. Cover every public API, config key, CLI flag,
  environment variable, endpoint, class/field.
- **Is austere and consistent.** A uniform structure across every entry so readers
  learn the shape once and then scan fast. Same headings, same order, every time.
- **Mirrors the structure of the code / product.** The documentation's architecture
  should follow the software's architecture, so a reader who knows where a thing
  lives in the product knows where to find its docs.
- **Is for looking up, not reading through.** Optimise for search and scan:
  predictable layout, tables, alphabetical or structural ordering, anchors.

## What reference must NOT do

- Don't explain concepts or rationale — link to **explanation**.
- Don't walk through tasks — link to **how-to guides**.
- Don't teach — link to **tutorials**.
- Don't editorialise ("we recommend…", "the best way is…") — that's explanation.
  Reference states the facts and lets other modes advise.

## The recurring temptation

Reference is where authors most often smuggle in the other modes, because the reader
is right there needing help. Resist it. A single example of *usage* per entry is fine
and often valuable, but a walkthrough is a how-to and belongs elsewhere. Keep
reference a reliable, boring, complete source of truth.

## Structure patterns

- **API / function reference:** name, signature, parameters (name, type, default,
  meaning), return value, raised errors, one minimal usage example.
- **Configuration reference:** key, type, default, allowed values, effect, since-version.
- **CLI reference:** command, synopsis, each flag with type/default/effect, exit codes.
- **Data model / schema reference:** entity, each field with type/constraints/meaning.

Group entries the way the product is organised; within a group, order predictably
(alphabetical, or by the code's own ordering). Provide a complete index.

## Writing checklist

- [ ] Every public surface is covered — no gaps.
- [ ] Uniform entry template applied to every item.
- [ ] Facts only: types, defaults, ranges, errors, effects.
- [ ] Structure mirrors the product; index/anchors make lookup fast.
- [ ] At most one small usage example per entry; no walkthroughs.
- [ ] Cross-link to how-tos and explanation instead of inlining them.
- [ ] Kept in sync with the code — stale reference is worse than none.
