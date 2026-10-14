---
name: shadcn-accessibility
description: Use when building, reviewing, testing, or advising on the accessibility of a web app — especially modern React/shadcn/ui apps that must meet WCAG 2.2 AA. Covers all 50 WCAG 2.2 Level A and AA success criteria (with shadcn/Radix-specific how-to-meet tips), how to test (automated axe/Playwright, manual, assistive technology), and how to write a voluntary accessibility statement / VPAT.
---

# Accessibility (WCAG 2.2 AA) for modern web apps

Help developers make shadcn/ui-based web apps accessible and meet **WCAG 2.2 Level AA** — the de-facto bar referenced by the EU Accessibility Act, US ADA/Section 508 (VPAT), and most procurement.

## Why WCAG 2.2 AA is the target

- **WCAG 2.2 Level AA** — every Level A and Level AA success criterion (50 in total). Level AAA is not required.
- It is the standard cited by the **EU Accessibility Act** (in force June 2025 for many private-sector digital products/services), **US ADA** case law and **Section 508 / VPAT** procurement, UK equality duties, and most enterprise buyers. Even where no law strictly applies to an SME app, AA is the expected quality bar.
- shadcn/ui is built on **Radix UI**, which gives a strong accessibility *baseline* (focus, keyboard, ARIA) — but a baseline is not a guarantee. Labels, contrast, alt text, headings, and accessible names are still on you.

## Bundled references — read the relevant one before answering in detail

| File | When to read it |
| --- | --- |
| `references/wcag-2.2.md` | Auditing or implementing against specific criteria. All 50 A/AA criteria grouped by POUR, each with a plain-English requirement and a how-to-meet tip (shadcn/Radix-aware where relevant). Flags the 6 new 2.2 criteria and the removal of 4.1.1 Parsing. |
| `references/testing.md` | Planning or running accessibility testing — automated tools (axe, jest-axe, Playwright, Storybook a11y, Lighthouse) and their limits, manual checks, assistive-technology pairings, and a pre-launch checklist. |
| `references/accessibility-statement.md` | Writing an accessibility statement or a VPAT/ACR. Includes a copy-and-adapt template and guidance on the EAA / ADA / Section 508 regimes. |

Distilled from W3C WCAG 2.2 and common practice (WCAG © W3C). For exact wording always confirm against <https://www.w3.org/TR/WCAG22/> and the [How to Meet WCAG quick reference](https://www.w3.org/WAI/WCAG22/quickref/).

## Working rules

- **Automated tools catch only ~30–40% of issues.** Never claim a page is accessible on the strength of axe/Lighthouse alone. Always combine automated checks with keyboard-only testing, zoom/reflow, and a screen-reader pass. See `references/testing.md`.
- **Radix is a baseline, not a guarantee.** Using `Dialog`/`Select`/`DropdownMenu` gives correct keyboard/focus/ARIA — but an unlabelled input, a titleless dialog, an icon-only button with no accessible name, or a low-contrast token still fails. Check the composition, not just the primitive.
- **Don't assert "WCAG 2.2 AA compliant" loosely.** Tie any claim to specific criteria and prefer "tested against…" over blanket compliance unless an audit backs it.
- When reviewing code, think in the four principles: is content **Perceivable** (text alternatives, contrast, structure), **Operable** (keyboard, focus, target size, no traps), **Understandable** (labels, errors, consistent help), and **Robust** (valid name/role/value, status messages)?
- Common high-impact fixes to check first: meaningful `alt` (and `alt=""` for decorative), every control labelled (`Label`/shadcn `Form`), visible `focus-visible:ring` never stripped, logical heading order, 4.5:1 text contrast (watch `text-muted-foreground` on tinted surfaces), `lang` on `<html>`, errors announced and linked, `DialogTitle` present, accessible names on icon-only buttons, 24×24px targets (2.5.8).

## Related

- `shadcn-design-system` skill — the components (built on Radix) that meet many of these criteria when used correctly.
- Commands: `/shadcn:audit`, `/shadcn:review`, `/shadcn:accessibility-statement`.
