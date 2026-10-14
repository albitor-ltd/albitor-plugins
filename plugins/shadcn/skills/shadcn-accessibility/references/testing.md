# Testing for accessibility

> **Sources:** [W3C WAI: Evaluating Web Accessibility](https://www.w3.org/WAI/test-evaluate/), [Deque axe-core](https://github.com/dequelabs/axe-core), and industry consensus (Deque, WebAIM, Level Access) on automated coverage.
> © W3C / respective authors. Distilled reference — verify against the live source for current detail. Tooling is oriented at a React/shadcn/ui stack.

## The headline rule

**Automated testing alone is not enough.** Automated tools reliably detect only about **30–40%** of WCAG issues. The remaining 60–70% — meaningful alt text, logical reading/focus order, sensible headings, understandable errors, keyboard operability of custom widgets, whether a Radix composition is actually usable — need **manual** and **assistive-technology** testing, and ultimately testing **with disabled users**.

A complete approach combines four layers: **automated → manual → assistive tech → real users.** Test continuously, not just before launch.

> **Radix caveat:** shadcn's Radix primitives pass most automated checks by construction — a green axe run on a Radix Dialog tells you little about whether *your* dialog has a title, labels, and sufficient contrast. Don't let the baseline lull you.

## 1. Automated tools

| Tool | Form | Good for |
| --- | --- | --- |
| **axe DevTools** | Browser extension | Fast, low-false-positive rule checks while developing. |
| **axe-core** | Library | The engine behind most of the below; wire into tests. |
| **jest-axe / vitest-axe** | Unit test matcher | Assert `toHaveNoViolations()` on rendered components (great for a shadcn component library). |
| **@axe-core/playwright** | E2E library | Run axe against real rendered pages in CI (`AxeBuilder(page).analyze()`). |
| **Storybook a11y addon** (`@storybook/addon-a11y`) | Storybook | Per-story axe panel — ideal if you document components in Storybook. |
| **Lighthouse** | Chrome DevTools / CI | Page-level accessibility score (uses axe rules) alongside performance. |
| **eslint-plugin-jsx-a11y** | Lint | Catches many JSX issues (missing `alt`, `onClick` on non-interactive, bad ARIA) at author time. |
| **pa11y** | CLI / CI | Scriptable command-line scanning across URLs. |

**What they catch:** missing `alt`, missing form labels, low colour contrast (static colours), missing `lang`, duplicate IDs, some ARIA misuse, empty links/buttons, `onClick` on non-interactive elements.

**What they miss:** whether `alt` is *meaningful*, whether reading/focus order makes sense, whether content is understandable, whether custom widgets work by keyboard/screen reader, contrast of colours set via CSS variables/theme, and most context-dependent criteria.

> Wire axe (via `@axe-core/playwright` or `jest-axe`) and `eslint-plugin-jsx-a11y` into CI so regressions are caught — but never treat a green run as "accessible".

## 2. Manual checks (no special software needed)

- **Keyboard-only navigation** — unplug the mouse. Tab/Shift+Tab through everything: every interactive element reachable, operable, with a **visible focus ring** (2.4.7). Focus order logical (2.4.3); no traps (2.1.2); focus not hidden behind sticky headers/`Sheet`s/cookie bars (2.4.11). Check the skip link (2.4.1). For Radix components verify `Esc` closes overlays and focus returns to the trigger.
- **Zoom to 400%** — at 1280px zoomed to 400% (≈320px CSS), content must **reflow** without horizontal scrolling or clipping (1.4.10, 1.4.4). Wide tables should scroll within their own container, not the page.
- **Text spacing** — apply the text-spacing bookmarklet; nothing should clip (1.4.12).
- **Colour contrast** — check text (4.5:1, or 3:1 large) and UI/graphics (3:1) with a contrast checker (1.4.3, 1.4.11), in **both light and dark themes**. The usual failures: `text-muted-foreground` on tinted cards, brand `primary-foreground` on `primary`, disabled `opacity-50` states, and the `--ring` colour. Check info isn't conveyed by colour alone (1.4.1).
- **Content structure / headings** — one logical `<h1>`, correct hierarchy; landmarks (`<nav>`, `<main>`, `<footer>`); real semantic markup for lists, tables (`<th scope>`), and form labels (1.3.1, 2.4.6). A headings-outline extension or the browser a11y inspector helps.
- **Forms** — every field has a persistent `Label`/`FormLabel`; errors described in text and suggest a fix; a review/confirm step for consequential submissions (3.3.1–3.3.4).
- **Reduced motion / autoplay** — animations respect `prefers-reduced-motion`; carousels/tickers can be paused (2.2.2); nothing flashes >3×/sec (2.3.1).
- **Target size** — pointer targets ≥ 24×24px, aim for ≥44px on primary touch targets (2.5.8) — check dense table row-actions and icon rows.
- **Link/button text** — makes sense out of context; icon-only buttons have an accessible name (2.4.4, 4.1.2).

## 3. Testing with assistive technology

Test with the combinations real users actually use:

| Assistive tech | Recommended browser | Platform |
| --- | --- | --- |
| **NVDA** (free) | **Firefox** (or Chrome) | Windows |
| **JAWS** | **Chrome** (or Edge) | Windows |
| **VoiceOver** | **Safari** | macOS / iOS |
| **TalkBack** | Chrome | Android |
| **Voice control** (Dragon, Voice Control, Voice Access) | — | Check spoken labels match visible labels (2.5.3) |
| **Screen magnifier** (ZoomText, OS magnifier) | — | Layout/focus tracking at high magnification |

When screen-reader testing a shadcn app, confirm: images announce sensible alternatives; headings/landmarks let you navigate; a `Dialog` announces its `DialogTitle` on open and returns focus on close; `Select`/`DropdownMenu`/`Tabs` announce role and state; form fields announce label, role, and error; toasts (`Sonner`) are announced via live region (4.1.3); custom widgets announce correct name/role/value (4.1.2).

> NVDA + Firefox and VoiceOver + Safari are the most common free starting points. JAWS + Chrome covers the dominant commercial screen reader.

## 4. Testing with disabled users and a formal audit

- **Test with disabled and older users** as part of user research — surfaces issues no tool or checklist will.
- **Get a formal audit** before a major launch or for VPAT/procurement — a WCAG 2.2 report from an accessibility specialist (e.g. Deque, TPGi, or a national provider) feeds your accessibility statement and remediation plan.

## Pre-launch checklist

- [ ] axe (via `@axe-core/playwright` and/or `jest-axe`) and `eslint-plugin-jsx-a11y` wired into CI and passing.
- [ ] Full keyboard-only pass: everything reachable, operable, visible focus, logical order, no traps, focus never obscured; overlays close on `Esc` and return focus.
- [ ] Skip link present and working; landmarks in place; single logical `<h1>` + correct hierarchy.
- [ ] Reflows cleanly at 400% / 320px; wide tables scroll internally; survives increased text spacing.
- [ ] Text contrast ≥ 4.5:1 (3:1 large); UI/graphic contrast ≥ 3:1 — **checked in both light and dark themes**; no colour-only information.
- [ ] Every form field labelled; errors identified, described, and suggest a fix; review step for key journeys.
- [ ] Every `Dialog`/`Sheet` has a `DialogTitle`; every icon-only button has an accessible name.
- [ ] Pointer targets ≥ 24×24px; drag/slider actions have a non-drag alternative; gesture alternatives exist.
- [ ] Screen-reader pass with NVDA+Firefox and VoiceOver+Safari (ideally JAWS+Chrome too).
- [ ] Dynamic updates announced (Sonner/live regions); custom widgets expose correct name/role/value.
- [ ] Page titles unique and descriptive; `lang` set on `<html>`; `prefers-reduced-motion` respected.
- [ ] Tested with disabled users; formal audit completed before a major/procurement launch.
- [ ] Accessibility statement published, accurate, and listing all known issues with fix dates.
