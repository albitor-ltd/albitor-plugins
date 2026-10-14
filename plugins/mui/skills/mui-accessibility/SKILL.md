---
name: mui-accessibility
description: Use when building, reviewing, testing, or advising on the accessibility of a Material UI (MUI) or general web app — meeting WCAG 2.2 AA. Covers all 50 WCAG 2.2 Level A and AA success criteria, how MUI components help you meet them (and how misusing them breaks them), how to test (automated with axe, manual, assistive technology), and a general-purpose accessibility-statement template.
---

# Accessibility (WCAG 2.2 AA) for Material UI apps

Help developers make consumer and general-purpose web apps accessible, using **WCAG 2.2 Level AA** as the conformance target. AA is the level referenced by the regimes most products meet — **EN 301 549** and the **European Accessibility Act**, **Section 508** and the **ADA** in the US — so building to WCAG 2.2 AA satisfies the common baseline. MUI components are accessible out of the box, but only if used correctly.

## What "accessible" means here (the short version)

- Meet **WCAG 2.2 Level AA** — every Level A and Level AA success criterion (50 in total). Level AAA is not required.
- **MUI's accessibility is a starting point, not a guarantee.** Components ship correct roles, focus management and keyboard support, but you still have to label them (a `TextField` `label`, an icon-`IconButton` `aria-label`, a `Dialog` `aria-labelledby`), keep contrast after theming, and not convey status by colour alone.
- If your product is sold or offered in a regulated market, publish an **accessibility statement** — see `references/accessibility-statement.md`.

## Bundled references — read the relevant one before answering in detail

| File | When to read it |
| --- | --- |
| `references/wcag-2.2.md` | Auditing or implementing against specific criteria. All 50 A/AA criteria grouped by POUR, each with a plain-English requirement and a how-to-meet tip (with MUI-specific notes). Flags the 6 new 2.2 criteria and the removal of 4.1.1 Parsing. |
| `references/testing.md` | Planning or running accessibility testing — automated tools (axe/Lighthouse) and their limits, manual checks, assistive-technology pairings, and a pre-launch checklist. |
| `references/accessibility-statement.md` | Writing or reviewing an accessibility statement for a consumer/general-purpose product, with a copy-and-adapt template. |

Distilled from W3C WCAG 2.2 and MUI's accessibility guidance (WCAG © W3C; MUI is MIT-licensed). For exact wording always confirm against <https://www.w3.org/TR/WCAG22/> and the MUI component API pages.

## Working rules

- **Automated tools catch only ~30–40% of issues.** Never claim a page is accessible on the strength of axe/Lighthouse alone. Always combine automated checks with keyboard-only testing, zoom/reflow, and a screen-reader pass. See `references/testing.md`.
- **Don't assert "WCAG 2.2 AA compliant" loosely.** Tie any claim to specific criteria, and prefer "tested against…" over blanket compliance unless an audit backs it.
- Think in the four principles: is content **Perceivable** (text alternatives, contrast, structure), **Operable** (keyboard, focus, target size, no traps), **Understandable** (labels, errors, consistent help), and **Robust** (valid name/role/value, status messages)?
- Common high-impact fixes to check first in an MUI app: `TextField` labels, icon-`IconButton` `aria-label`, meaningful `Avatar`/image `alt`, visible focus (don't strip MUI's), logical heading order (`Typography component=…`), 4.5:1 text contrast after theming, `lang` on `<html>`, errors announced and linked (`error` + `helperText`), `Dialog` labelling, and 24×24px target sizes (2.5.8).
- MUI components help meet many criteria — but only if used correctly and not fought with `!important` or replaced by hand-rolled `<div onClick>` controls. Pair with the `mui-design-system` skill.

## Related

- `mui-design-system` skill — the accessible components that meet many of these criteria out of the box.
- Commands: `/mui:audit`, `/mui:accessibility-statement`, `/mui:review`.
