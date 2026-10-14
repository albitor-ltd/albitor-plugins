---
name: wcag-22-aa
description: The WCAG 2.2 Level A and AA success criteria and how to test them, independent of any design system. Use when writing or reviewing an accessibility check, a journey that audits a screen, or a screen's markup in an app composed from the test-playwright harness.
---

# WCAG 2.2 A and AA, framework-neutral

The Internal Tool governance profile holds an app to **every WCAG 2.2 Level A and AA criterion** (50:
31 A and 19 AA). Level AAA is not asked for. This skill is the shared reference; each design system's
own accessibility skill adds what its components do for you.

| File | Read it when |
| --- | --- |
| `references/wcag-2.2.md` | Implementing or auditing a specific criterion: all 50, grouped by principle, each with the requirement and a how-to-meet tip. Flags the six criteria new in 2.2 and the removal of 4.1.1 Parsing |
| `references/testing.md` | Planning accessibility testing: what automated tools find and miss, manual checks, assistive-technology pairings, a pre-launch checklist |

Distilled from W3C WCAG 2.2 (© W3C, W3C Document Licence). Confirm exact wording against
<https://www.w3.org/TR/WCAG22/>.

## What the harness checks, and what it does not

- `web/tests/helpers/a11y.ts` runs **axe-core** against exactly the A and AA tags of 2.0, 2.1 and
  2.2 (`wcag2a`, `wcag2aa`, `wcag21a`, `wcag21aa`, `wcag22aa`). `web/tests/a11y.spec.ts` audits the
  shell's own screens; a journey audits the screens it walks with
  `journeyStep(page, name, body, { a11y: true })`.
- Automated tools find roughly 30 to 40 percent of problems. A green audit means "no automatable
  failure", never "accessible". Reading order, text alternatives that mean something, error wording,
  keyboard operation of a custom widget and zoom or reflow still need a person.
- The audit sees what is rendered. Wait for the screen to settle (assert something the person must
  see) before auditing, or it audits a blank page and passes.
- A waiver switches one rule off for one audit and **must carry a reason**:
  `expectNoA11yViolations(page, { waive: [{ rule: "color-contrast", reason: "third-party map tiles" }] })`.
  Prefer `exclude` of the one third-party element over waiving a rule for the whole page.
- Do not state "WCAG 2.2 AA compliant" on the strength of axe. Tie a claim to the criteria tested.
