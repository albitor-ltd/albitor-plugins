---
description: Draft an accessibility statement for a consumer or general-purpose web app.
argument-hint: "[product name / details, or leave blank to be prompted]"
---

Help the user produce a clear, honest accessibility statement for a consumer or general-purpose product. Context provided: **$ARGUMENTS**

1. Load the `mui-accessibility` skill and read `references/accessibility-statement.md` (template + guidance on what to claim).
2. Gather the facts you need — ask concisely for anything missing:
   - Product/website name and URL, and the responsible organisation.
   - Conformance status against **WCAG 2.2 AA**: **fully**, **partially**, or **not yet** conformant (be honest — most products are "partially").
   - Known non-accessible areas and the failing criteria (offer to run `/mui:audit` first if unknown).
   - Which regime, if any, applies (e.g. the **European Accessibility Act / EN 301 549** for products sold in the EU, **ADA / Section 508** in the US) — this shapes tone and whether a formal conformance claim is expected.
   - Contact route for accessibility problems and a response-time commitment, plus how to request an alternative.
   - Date prepared and how it was tested (self-assessment with axe + manual/AT testing, or an external audit).
3. Produce the statement by filling the template: what the product does to be accessible, the conformance status tied to specific WCAG 2.2 AA criteria, known issues with target fix dates, feedback/contact route, and the testing basis.
4. Flag anything asserted but unverified (e.g. "fully conformant" without an audit) and recommend the testing needed to back it. Prefer "tested against WCAG 2.2 AA using…" over a blanket compliance claim unless an audit supports it.

Output the finished statement as clean Markdown the user can publish, plus a short note of what still needs confirming.
