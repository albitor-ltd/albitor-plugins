# Testing for accessibility

> **Sources:** [GDS Service Manual: Testing for accessibility](https://www.gov.uk/service-manual/helping-people-to-use-your-service/testing-for-accessibility) and [GDS: Understanding WCAG](https://www.gov.uk/service-manual/helping-people-to-use-your-service/understanding-wcag). Automated-coverage figure corroborated by industry consensus (Deque, Level Access, WebAIM).
> Crown Copyright / W3C. GOV.UK guidance licensed under the Open Government Licence v3.0; WCAG 2.2 © W3C. Distilled reference — verify against the live source for current detail.

## The headline rule

**Automated testing alone is not enough.** Automated tools reliably detect only about **30–40%** of WCAG issues (estimates range ~20–40%). The remaining 60–70% — meaningful alt text, logical reading/focus order, sensible headings, understandable error messages, keyboard operability of custom widgets — need **manual** and **assistive-technology** testing, and ultimately testing **with disabled users**.

A complete approach combines four layers: **automated → manual → assistive tech → real users.** Test continuously from alpha/beta onward, not just before launch.

## 1. Automated tools

| Tool | Form | Good for |
| --- | --- | --- |
| **axe** (axe-core / axe DevTools) | Browser extension + CI library | Fast, low false-positive rule checks; integrates into unit/E2E tests. |
| **Lighthouse** | Chrome DevTools / CI | Quick page-level accessibility score (uses axe rules) alongside performance. |
| **WAVE** | Browser extension | Visual overlay of errors, contrast, structure and ARIA on the page. |
| **pa11y** | CLI / CI | Scriptable command-line scanning across many URLs; good for pipelines. |
| **ARC Toolkit / SiteImprove** | Extension / platform | Detailed rule reporting; site-wide monitoring (SiteImprove). |

**What they catch:** missing `alt`, missing form labels, low colour contrast, missing `lang`, duplicate IDs, some ARIA misuse, empty links/buttons.

**What they miss:** whether alt text is *meaningful*, whether reading/focus order makes sense, whether content is understandable, whether custom widgets actually work by keyboard or screen reader, and most context-dependent criteria.

> Build axe or pa11y into CI so regressions are caught automatically — but never treat a green automated run as "accessible".

## 2. Manual checks (no special software needed)

- **Keyboard-only navigation** — unplug the mouse. Tab/Shift+Tab through everything: every interactive element must be reachable, operable, and have a **visible focus indicator** (2.4.7); focus order must be logical (2.4.3); no keyboard traps (2.1.2); focus must not be hidden behind sticky headers/cookie banners (2.4.11). Check skip links work (2.4.1).
- **Zoom to 400%** — at 1280px width zoomed to 400% (≈320px CSS), content must **reflow** without horizontal scrolling or clipping (1.4.10, 1.4.4).
- **Text spacing** — apply a bookmarklet that increases line/letter/word/paragraph spacing; nothing should be cut off (1.4.12).
- **Colour contrast** — check text (4.5:1, or 3:1 large) and UI/graphics (3:1) with a contrast checker (1.4.3, 1.4.11). Check information isn't conveyed by colour alone (1.4.1).
- **Content structure / headings** — verify a single logical `<h1>` and a correct heading hierarchy; check landmarks (`<nav>`, `<main>`, `<footer>`); confirm lists, tables and form labels use real semantic markup (1.3.1, 2.4.6). A browser accessibility inspector (Firefox Accessibility Inspector, Chrome DevTools) or a headings-outline extension helps.
- **Forms** — every field has a persistent label, errors are described in text and suggest a fix, and there's a "check your answers" step for important submissions (3.3.1–3.3.4).
- **Reduced motion / autoplay** — moving content can be paused (2.2.2); nothing flashes more than 3×/sec (2.3.1).
- **Target size** — pointer targets are at least 24×24px (aim for 44×44px) (2.5.8).
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

When screen-reader testing, confirm: images announce sensible alternatives; headings and landmarks let you navigate; form fields announce their label, role and error state; dynamic updates are announced via live regions (4.1.3); custom widgets announce correct name/role/value (4.1.2).

> NVDA + Firefox and VoiceOver + Safari are the most common free starting points. JAWS + Chrome covers the dominant commercial screen reader.

## 4. Testing with disabled users and a formal audit

- **Test with disabled and older users** as part of user research — this is required by the Service Standard and surfaces issues no tool or checklist will.
- **Get a formal accessibility audit** before your service enters **public beta**. The leading UK provider is the **Digital Accessibility Centre (DAC)**, which audits with disabled testers and assistive tech and produces a WCAG 2.2 report; the results feed your accessibility statement and remediation plan.

## Pre-launch checklist

- [ ] Automated scan (axe/Lighthouse/WAVE/pa11y) passes with no outstanding errors, **and** is wired into CI.
- [ ] Full keyboard-only pass: everything reachable, operable, visible focus, logical order, no traps, focus never obscured.
- [ ] Skip link present and working; landmarks in place.
- [ ] Reflows cleanly at 400% zoom / 320px; survives increased text spacing.
- [ ] Text contrast ≥ 4.5:1 (3:1 large); UI/graphic contrast ≥ 3:1; no colour-only information.
- [ ] One logical `<h1>`, correct heading hierarchy, semantic markup throughout.
- [ ] Every form field labelled; errors identified, described and suggest a fix; "check your answers" step for key journeys.
- [ ] Pointer targets ≥ 24×24px; drag actions have a non-drag alternative; gesture alternatives exist.
- [ ] Screen-reader pass with NVDA+Firefox and VoiceOver+Safari (ideally JAWS+Chrome too).
- [ ] Dynamic updates announced (live regions); custom widgets expose correct name/role/value.
- [ ] Page titles unique and descriptive; `lang` set on `<html>`.
- [ ] Tested with disabled users; formal DAC (or equivalent) audit completed before public beta.
- [ ] Accessibility statement published, accurate, and listing all known issues with fix dates.
