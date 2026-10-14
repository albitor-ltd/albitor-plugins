---
description: Draft a voluntary accessibility statement (VPAT / ACR style) for a web app.
argument-hint: "[product name / details, or leave blank to be prompted]"
---

Help the user produce an honest, public accessibility statement for their web app. Unlike a UK public sector service, a commercial or internal SME app is usually **not** legally required to publish a prescribed statement — but a clear, specific statement is good practice, is increasingly expected (EU Accessibility Act, ADA, Section 508 / VPAT procurement), and builds trust. Context provided: **$ARGUMENTS**

1. Load the `shadcn-accessibility` skill and read `references/accessibility-statement.md` (template + guidance on VPAT/ACR and the relevant regimes).
2. Gather the facts you need — ask concisely for anything missing:
   - Product/site name and URL, and the responsible organisation.
   - Target standard and conformance level (default and recommended: **WCAG 2.2 AA**).
   - Conformance claim: **fully**, **partially**, or **not yet** conformant — and be honest; "fully" needs an audit behind it.
   - Known non-accessible content / features and the failing criteria (offer to run `/shadcn:audit` first if unknown).
   - Contact route for accessibility problems and how to request alternatives, with a response commitment.
   - Which regimes apply (EU EAA, US ADA/Section 508 VPAT, other) so the statement names the right standard and audience.
   - Date prepared and how it was tested (self-assessment with axe + manual/AT, or an external audit).
3. Produce the statement from the template: an overview of what users can do, the conformance claim tied to **WCAG 2.2 AA** (never a vague "we care about accessibility" with nothing behind it), a list of known issues each citing the WCAG success criterion it fails with a target fix date, feedback/contact details, and a "how this was prepared/tested" section. Offer a companion **VPAT/ACR** row-by-row table if procurement needs one.
4. Flag anything asserted but unverified (e.g. "fully conformant" without an audit) and recommend the testing needed to back it.

Output the finished statement as clean Markdown the user can publish, plus a short note of what still needs confirming.
