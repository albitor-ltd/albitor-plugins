---
description: Draft a GOV.UK-format accessibility statement for a public sector service.
argument-hint: "[service name / details, or leave blank to be prompted]"
---

Help the user produce a legally-compliant accessibility statement. Context provided: **$ARGUMENTS**

1. Load the `govuk-accessibility` skill and read `references/accessibility-statement.md` (template + required wording) and `references/regulations.md` (obligations).
2. Gather the facts you need — ask concisely for anything missing:
   - Service/website name and URL, and the responsible organisation.
   - Compliance status: **fully**, **partially**, or **not** compliant with WCAG 2.2 AA.
   - Known non-accessible content and the failing criteria (offer to run `/gds:audit` first if unknown).
   - Any disproportionate-burden or out-of-scope claims (and note these must be justified).
   - Contact route for accessibility problems and how to request alternative formats.
   - Date the statement was prepared and how the service was tested (self-assessment or external audit, e.g. DAC).
3. Produce the statement by filling the GDS model template, keeping the **legally-required enforcement-procedure paragraph verbatim** (use the Northern Ireland / ECNI variant if the body is NI-based).
4. Flag anything that is asserted but unverified (e.g. "fully compliant" without an audit) and recommend testing to back it.

Output the finished statement as clean Markdown the user can publish, plus a short note of what still needs confirming.
