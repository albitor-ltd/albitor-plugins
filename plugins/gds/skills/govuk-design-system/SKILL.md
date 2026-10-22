---
name: govuk-design-system
description: Use when building, reviewing, or discussing a GOV.UK or UK public sector frontend — any HTML/Nunjucks/React/Svelte UI that should follow the GOV.UK Design System. Covers the 34 components, page and "ask users for information" patterns, the GOV.UK colour/typography/layout/spacing styles, and how to consume GOV.UK Frontend (Nunjucks macros or copied HTML) with progressive enhancement.
metadata:
  capability: design-system
---

# GOV.UK Design System

Help developers build interfaces that follow the GOV.UK Design System so they follow GOV.UK Design System patterns and pass a service assessment, with branding per the stated on/not-on GOV.UK answer.

## First: is this service on GOV.UK? (decides what you may use)

GOV.UK branding follows one decision, and that decision is **stated explicitly by the build**: whether the service is published on GOV.UK (`*.service.gov.uk`). Albitor states it in the build prompt as "The person also said where the service will be published: on GOV.UK, on a `service.gov.uk` domain" or "…: not on GOV.UK: it is an internal service, or one on another domain". The domain wording inside that sentence is the statement itself, not an inference; only a domain you guess from elsewhere (a URL, the audience) is an inference. Getting it wrong is a Crown-branding and font-licensing breach, not a style nit.

- **Apply GOV.UK branding only when the build explicitly states the service is on GOV.UK.**
- **If the build states "not on GOV.UK", or does not state it at all, use the crownless pattern**: no crown, no GOV.UK logotype, no OGL, no Crown copyright, no GDS Transport. Follow the GOV.UK Frontend guidance for services not on GOV.UK: your organisation's name in the header (the service name goes in Service navigation, not the header), and the Arial font stack: `$govuk-font-family: arial, sans-serif` with `$govuk-include-default-font-face: false`.
- **Never infer it** from the domain, the audience, the organisation type or the words "public sector", "council", "NHS" or "government". An internal council tool on a CloudFront URL is crownless unless the build says it is on GOV.UK. If the statement is missing, default to crownless; do not guess and do not add branding.

| | **Stated on GOV.UK** (`*.service.gov.uk`) | **Stated not on GOV.UK, or not stated** |
| --- | --- | --- |
| Crown logo / GOV.UK logotype | ✅ Use it | ❌ **Never** — and don't imply you're part of GOV.UK or use the "GOV.UK" name |
| GOV.UK header & footer components | ✅ Use the branded ones (keep OGL + Crown copyright) | ❌ Don't use them — build your **own** header/footer |
| GDS Transport typeface | ✅ Use it (serve from your own assets) | ❌ Licensed for GOV.UK only — fall back to **Arial** (`$govuk-font-family: arial, sans-serif`) |
| Brand colour | ✅ Keep the GOV.UK brand colour | ✅ Use your **own** brand colour and logo |
| Components, patterns, grid/layout, spacing & type scale, functional colours (error red, focus yellow, link/button states) | ✅ Use them all | ✅ Use them all — the Design System is required *minus* the branding |

The rule "if not on GOV.UK you must still use the GOV.UK Design System, with the exception of branding (logos and fonts)" is mandated, not advisory. Sources: [Making your service look like GOV.UK](https://www.gov.uk/service-manual/design/making-your-service-look-like-govuk), [Use GOV.UK domains and the GOV.UK design system](https://www.gov.uk/guidance/use-govuk-domains-and-the-govuk-design-system), [GOV.UK Brand Guidelines](https://brand.design-system.service.gov.uk/).

## Core principles (apply these first)

1. **Use a component if one exists.** The Design System has a vetted, accessible component for almost every common need. Reach for it before inventing markup. The full catalogue is in `references/components.md`.
2. **Do not restyle components.** Keep the `govuk-*` classes and their meaning. Don't recolour buttons, change input borders, or repurpose the red error styling. Consistency across government is the point. (The one permitted change *off* GOV.UK is the brand colour — see the table above.)
3. **Build with progressive enhancement.** The non-JavaScript HTML must work on its own; JS only enhances it. Server-side validation is mandatory even when client-side validation exists.
4. **Compose with patterns, not just components.** A "question page", "check your answers" page, or "ask for an address" flow is a *pattern* — a tested arrangement of components and content. See `references/patterns.md`.
5. **Accessibility is not optional.** Every choice here exists partly for accessibility. Pair this with the `govuk-accessibility` skill when auditing.

## Bundled references — read the relevant one before answering in detail

| File | When to read it |
| --- | --- |
| `references/components.md` | Choosing/implementing any UI element. Lists all 34 components, their Nunjucks macro (`govuk*`), whether they need JS, when (not) to use them, and accessibility gotchas. |
| `references/patterns.md` | Designing a page or a flow, or asking users for information (names, addresses, dates, NINo, passwords, etc.). |
| `references/styles.md` | Colour, typography, the grid/layout, spacing scale, the page template, links and focus states. |
| `references/frontend-conventions.md` | Installing/using GOV.UK Frontend, Nunjucks vs plain HTML, `initAll`, the page template, and consuming the markup from a React/Svelte/other stack. |

These are distilled from the live GOV.UK Design System and GOV.UK Frontend (Crown Copyright, Open Government Licence v3.0). They are a fast index — for exact, current detail always confirm against <https://design-system.service.gov.uk/> and <https://frontend.design-system.service.gov.uk/>, especially colours (the 2025 GOV.UK brand refresh changed several functional colour values).

## Working rules

- **Don't hardcode hex colours.** Use the `govuk-colour()` Sass function or the functional colour helpers so the build stays correct across brand updates. See `references/styles.md`.
- **Prefer macros where the stack allows.** In Nunjucks/Node, use the official `govuk-frontend` macros — they keep markup correct as the library updates. In React/Svelte/etc., copy the component's HTML, keep every class, and port the JS behaviour (focus management, toggles) faithfully; note this is a maintenance cost you take on.
- **Forms:** every input needs an associated `<label>` (or `<legend>` in a `<fieldset>` for grouped controls). Never use placeholder text as a label. On error, render the error summary at the top (focused on load), link each entry to its field, and add the inline error message + error modifier classes.
- **Page furniture:** `govuk-width-container` and a phase banner (Alpha/Beta) apply to any service. The **branded GOV.UK header and footer** (crown, OGL/licence links) are for services **the build states are on GOV.UK only** — off GOV.UK, build your own header/footer (see the table above).
- When the user is starting a service, suggest the official **GOV.UK Prototype Kit** for prototyping but remind them production code should use `govuk-frontend` directly.

## Related

- `govuk-accessibility` skill — WCAG 2.2 AA and the legal regulations these components help you meet.
- Commands: `/gds:component`, `/gds:review`.
