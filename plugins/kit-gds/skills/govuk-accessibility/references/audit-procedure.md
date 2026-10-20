# Audit procedure: a WCAG 2.2 AA review

Run a WCAG 2.2 AA accessibility audit of: **the target the user names** (if none, the frontend files in the current changes).

Delegate to a fresh reviewer for a thorough pass, or do it inline for a small scope. Either way:

1. Load the `govuk-accessibility` skill; use `references/wcag-2.2.md` as the criteria checklist and `references/testing.md` for method.
2. Determine the target: a local file (Read it), a described component, or the current diff. State clearly what you examined and acknowledge what you **cannot** verify statically (focus order in a real browser, screen-reader output, contrast of dynamically-set colours) — recommend the manual/AT tests from `references/testing.md` for those.
3. Walk the four POUR principles. For each issue found, report:
   - **Criterion** (e.g. `1.1.1 Non-text Content`, Level A/AA)
   - **Severity** (Blocker / Major / Minor)
   - **Location** (`file:line`)
   - **What's wrong** and **the fix** (concrete code).
4. Prioritise: list Blockers and Majors first. Group the high-frequency checks (alt text, labels, focus visibility, heading order, contrast, `lang`, announced errors, 24px target size).
5. End with a short summary: counts by severity, and which criteria you could and could not assess statically.

Do not declare the page "WCAG 2.2 AA compliant" — automated/static review alone cannot establish that. Frame the result as findings plus the remaining manual tests needed.
