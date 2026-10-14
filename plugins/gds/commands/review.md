---
description: Review frontend changes against both the GOV.UK Design System and WCAG 2.2 AA.
argument-hint: "[file/dir path, or leave blank for the current changes]"
---

Review the GOV.UK frontend in scope: **$ARGUMENTS** (if empty, review the current uncommitted changes; fall back to recently edited frontend files).

Load **both** skills: `govuk-design-system` and `govuk-accessibility`. Then review on two axes:

1. **Design System conformance** (`references/components.md`, `patterns.md`, `styles.md`):
   - Are vetted components used where one exists, instead of bespoke markup?
   - Are components used unmodified (classes intact, colours/error styling not repurposed)?
   - Are the right patterns applied (question pages, check answers, error summary placement, page furniture)?
   - Hardcoded hex colours instead of `govuk-colour()` / Design System classes?

2. **Accessibility** (`references/wcag-2.2.md`): run the WCAG 2.2 AA checks — alt text, labels/legends, focus visibility, heading order, contrast, `lang`, announced + linked errors, target size, keyboard operability, progressive enhancement.

Report findings grouped by axis, each with `file:line`, severity (Blocker/Major/Minor), the problem, and a concrete fix. Note any issue that needs a real-browser or assistive-tech check rather than static review. Finish with a prioritised to-do list. Be specific and code-level; don't restate the standards in the abstract.
