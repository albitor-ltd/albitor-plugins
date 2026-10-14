# Testing for accessibility

> **Sources:** [W3C WAI: Evaluating Web Accessibility](https://www.w3.org/WAI/test-evaluate/) and [MUI accessibility guidance](https://mui.com/). Automated-coverage figure corroborated by industry consensus (Deque, Level Access, WebAIM).
> WCAG 2.2 © W3C. Distilled reference — verify against the live source for current detail.

## The headline rule

**Automated testing alone is not enough.** Automated tools reliably detect only about **30–40%** of WCAG issues (estimates range ~20–40%). The remaining 60–70% — meaningful alt text, logical reading/focus order, sensible headings, understandable error messages, keyboard operability of custom widgets — need **manual** and **assistive-technology** testing, and ultimately testing **with disabled users**.

**MUI is not a substitute for testing.** MUI components ship correct roles, focus management and keyboard support, but an app can still fail: an unlabelled `TextField`, an icon-`IconButton` with no `aria-label`, insufficient contrast after theming, colour-only status, a hand-rolled `<div onClick>`, or a broken heading order. Test the app you built, not the library.

A complete approach combines four layers: **automated → manual → assistive tech → real users.** Test continuously from the start, not just before launch.

## 1. Automated tools

| Tool | Form | Good for |
| --- | --- | --- |
| **axe** (axe-core / axe DevTools) | Browser extension + CI library | Fast, low false-positive rule checks; integrates into unit/E2E tests. The de facto baseline. |
| **Lighthouse** | Chrome DevTools / CI | Quick page-level accessibility score (uses axe rules) alongside performance. |
| **WAVE** | Browser extension | Visual overlay of errors, contrast, structure and ARIA on the page. |
| **pa11y** | CLI / CI | Scriptable command-line scanning across many URLs; good for pipelines. |
| **jest-axe / @axe-core/playwright** | Test library | Assert no violations in component/E2E tests — catches regressions on every PR. |

**What they catch:** missing `alt`, missing form labels, low colour contrast, missing `lang`, duplicate IDs, some ARIA misuse, empty links/buttons.

**What they miss:** whether alt text is *meaningful*, whether reading/focus order makes sense, whether content is understandable, whether custom widgets actually work by keyboard or screen reader, and most context-dependent criteria.

> Wire axe (jest-axe or @axe-core/playwright) into CI so regressions are caught automatically — but never treat a green automated run as "accessible".

## 2. Manual checks (no special software needed)

- **Keyboard-only navigation** — unplug the mouse. Tab/Shift+Tab through everything: every interactive element must be reachable, operable, and have a **visible focus indicator** (2.4.7); focus order must be logical (2.4.3); no keyboard traps (2.1.2); focus must not be hidden behind a fixed `AppBar`/sticky bar (2.4.11). Check skip links work (2.4.1). Verify `Dialog`/`Drawer`/`Menu` trap and restore focus and close on Esc.
- **Zoom to 400%** — at 1280px width zoomed to 400% (≈320px CSS), content must **reflow** without horizontal scrolling or clipping (1.4.10, 1.4.4).
- **Text spacing** — apply a bookmarklet that increases line/letter/word/paragraph spacing; nothing should be cut off (1.4.12).
- **Colour contrast** — check text (4.5:1, or 3:1 large) and UI/graphics (3:1) with a contrast checker (1.4.3, 1.4.11), **re-checking after theming/branding**. Check information isn't conveyed by colour alone (1.4.1) — e.g. an error is red **and** has text.
- **Content structure / headings** — verify a single logical `<h1>` and correct heading hierarchy (watch `Typography variant` vs `component` — visual size must not scramble semantics); check landmarks (`<nav>`, `<main>`, `<footer>`); confirm lists, tables and form labels use real semantic markup (1.3.1, 2.4.6).
- **Forms** — every field has a persistent label, errors are described in text (`error` + `helperText`) and suggest a fix, and there's a review step for important submissions (3.3.1–3.3.4).
- **Reduced motion / autoplay** — moving content can be paused (2.2.2); respect `prefers-reduced-motion`; nothing flashes more than 3×/sec (2.3.1).
- **Target size** — pointer targets are at least 24×24px (aim for 44×44px); watch `size="small"` `IconButton`/`Chip` (2.5.8).
- **Link text** — links make sense out of context; replace "click here"/"read more" (2.4.4).

## 3. Testing with assistive technology

Test with the combinations real users actually use. Recommended pairings:

| Assistive tech | Recommended browser | Platform |
| --- | --- | --- |
| **NVDA** (free) | **Firefox** (or Chrome) | Windows |
| **JAWS** | **Chrome** (or Edge) | Windows |
| **VoiceOver** | **Safari** | macOS / iOS |
| **TalkBack** | Chrome | Android |
| **Voice control** (Dragon, Voice Control, Voice Access) | — | Test that spoken labels match visible labels (2.5.3) |
| **Screen magnifier** (ZoomText, OS magnifier) | — | Check layout and focus tracking at high magnification |

When screen-reader testing an MUI app, confirm: images/`Avatar`s announce sensible alternatives; headings and landmarks let you navigate; `TextField`s announce their label, role and error state; `Dialog`s announce their title; dynamic updates (`Snackbar`/`Alert`) are announced via live regions (4.1.3); menus/tabs/tree views announce correct name/role/value (4.1.2).

> NVDA + Firefox and VoiceOver + Safari are the most common free starting points. JAWS + Chrome covers the dominant commercial screen reader.

## 4. Testing with disabled users and a formal audit

- **Test with disabled and older users** as part of user research — this surfaces issues no tool or checklist will.
- **Get a formal accessibility audit** before a major launch, especially if a regime applies (European Accessibility Act / EN 301 549, ADA, Section 508). An auditor tests with disabled testers and assistive tech and produces a WCAG 2.2 report that feeds your accessibility statement and remediation plan.

## Pre-launch checklist

- [ ] Automated scan (axe/Lighthouse/WAVE/pa11y) passes with no outstanding errors, **and** axe is wired into CI (jest-axe / @axe-core/playwright).
- [ ] Full keyboard-only pass: everything reachable, operable, visible focus, logical order, no traps, focus never obscured; dialogs/drawers/menus trap and restore focus.
- [ ] Skip link present and working; landmarks in place.
- [ ] Reflows cleanly at 400% zoom / 320px; survives increased text spacing.
- [ ] Text contrast ≥ 4.5:1 (3:1 large); UI/graphic contrast ≥ 3:1; **re-checked after theming**; no colour-only information.
- [ ] One logical `<h1>`, correct heading hierarchy (`Typography component` set correctly), semantic markup throughout.
- [ ] Every `TextField` labelled; errors identified, described and suggest a fix; review step for key journeys.
- [ ] Every icon-only `IconButton`/`Fab` has an `aria-label`; every `Dialog` is labelled.
- [ ] Pointer targets ≥ 24×24px; drag actions have a non-drag alternative; gesture alternatives exist.
- [ ] Screen-reader pass with NVDA+Firefox and VoiceOver+Safari (ideally JAWS+Chrome too).
- [ ] Dynamic updates announced (live regions); custom widgets expose correct name/role/value.
- [ ] Page titles unique and descriptive; `lang` set on `<html>`.
- [ ] Tested with disabled users; formal audit completed if a regime applies.
- [ ] Accessibility statement published, accurate, and listing known issues with fix dates.
