---
name: govuk-accessibility
description: Use when building, reviewing, testing, or advising on the accessibility of a website or app — especially UK public sector services that must meet WCAG 2.2 AA by law. Covers all 50 WCAG 2.2 Level A and AA success criteria, the Public Sector Bodies Accessibility Regulations 2018, the mandatory accessibility statement, and how to test (automated, manual, assistive technology).
---

# Accessibility (WCAG 2.2 AA) for UK public sector

Help developers make services accessible and meet the legal requirements that apply to UK public sector websites and apps.

## What the law requires (the short version)

- UK public sector bodies must meet **WCAG 2.2 Level AA** — every Level A and Level AA success criterion. Level AAA is not required.
- They must **publish and keep up to date an accessibility statement** in a prescribed format.
- This sits on top of the **Equality Act 2010** duty not to discriminate; the regulations make WCAG the concrete benchmark. Enforcement is by the EHRC/ECNI and the EASS; monitoring is by the Cabinet Office.
- Details, dates, scope and exemptions are in `references/regulations.md`.

## Bundled references — read the relevant one before answering in detail

| File | When to read it |
| --- | --- |
| `references/wcag-2.2.md` | Auditing or implementing against specific criteria. All 50 A/AA criteria grouped by POUR, each with a plain-English requirement and a how-to-meet tip. Flags the 6 new 2.2 criteria and the removal of 4.1.1 Parsing. |
| `references/regulations.md` | Questions about who must comply, deadlines, exemptions (disproportionate burden, third-party/archived content), monitoring and enforcement. |
| `references/accessibility-statement.md` | Writing or reviewing an accessibility statement. Includes a copy-and-adapt GDS model template and the legally-required enforcement wording. |
| `references/testing.md` | Planning or running accessibility testing — automated tools and their limits, manual checks, assistive-technology pairings, and a pre-launch checklist. |

Distilled from W3C WCAG 2.2 and GOV.UK guidance (WCAG © W3C; GOV.UK guidance under the Open Government Licence v3.0). For exact wording always confirm against <https://www.w3.org/TR/WCAG22/> and the relevant GOV.UK guidance page.

## Working rules

- **Automated tools catch only ~30–40% of issues.** Never claim a page is accessible on the strength of axe/Lighthouse alone. Always combine automated checks with keyboard-only testing, zoom/reflow, and a screen-reader pass. See `references/testing.md`.
- **Don't assert "WCAG 2.2 AA compliant" loosely.** Tie any claim to specific criteria, and prefer "tested against…" over blanket compliance statements unless an audit backs it.
- When reviewing code, think in terms of the four principles: is content **Perceivable** (text alternatives, contrast, structure), **Operable** (keyboard, focus, target size, no traps), **Understandable** (labels, errors, consistent help), and **Robust** (valid name/role/value, status messages)?
- Common high-impact fixes to check first: meaningful `alt` text (and `alt=""` for decorative images), every form control labelled, visible focus indicators, logical heading order, 4.5:1 text contrast, `lang` on `<html>`, error messages that are announced and linked, and 24×24px minimum target sizes (2.5.8, new in 2.2).
- The GOV.UK Design System components are built to help meet many of these criteria — but they only help if used correctly and not restyled. Pair with the `govuk-design-system` skill.

## Related

- `govuk-design-system` skill — accessible components that meet many of these criteria out of the box.
- Commands: `/gds:audit`, `/gds:accessibility-statement`, `/gds:review`.
