Source: GOV.UK Design System (https://design-system.service.gov.uk/) and GOV.UK Frontend. Crown Copyright, licensed under the Open Government Licence v3.0. Distilled reference — verify against the live source for current detail.

# GOV.UK Design System styles

Covers colour, typography, layout/grid, spacing, the page template, links and focus states.

Golden rule (stated on the styles overview): do not assign new meanings to colours, do not change the style of buttons, and do not adjust the thickness of borders on form inputs. Always consume colours via Sass functions (`govuk-colour()` / functional colour helpers), never hardcoded hex — this keeps services aligned when the palette changes.

> Note on the 2025 GOV.UK brand refresh: GOV.UK rebranded in 2025 and GOV.UK Frontend's functional colours were updated. Some values below differ from the long-standing pre-rebrand ones (e.g. link colour, error red, button green). The hex values are given for reference only — use the Sass functions in real code. Verify the exact current values against the live colour page.

## Colour

### How to use colour in code
- Full web palette: `govuk-colour("blue")` returns Primary blue `#1d70b8`. Pass a variant for tints/shades where supported.
- Functional (semantic) colours: use the functional colour helper (e.g. `govuk-functional-colour("brand")`) rather than the raw hex. The docs explicitly say: use `govuk-functional-colour("brand")` rather than `#1d70b8`.

### Functional / semantic colours (as fetched May 2026; verify against source)
| Role | Value |
| --- | --- |
| Text | `#0b0c0c` |
| Secondary text | `#484949` |
| Brand | `#1d70b8` |
| Link | `#1a65a6` |
| Link hover | `#0f385c` |
| Link visited | `#54319f` |
| Link active | `#0b0c0c` |
| Focus | `#ffdd00` |
| Focus text | `#0b0c0c` |
| Error | `#ca3535` |
| Success | `#0f7a52` |
| Hover (e.g. row) | `#cecece` |
| Border | `#cecece` |
| Input border | `#0b0c0c` |
| Body background | `#ffffff` |
| Template / surface background | `#f4f8fb` |
| Surface text | `#0b0c0c` |
| Surface border | `#8eb8dc` |

The web palette (`govuk-colour()`) groups: Blue, Green, Teal, Purple, Magenta, Red, Orange, Yellow, Brown, Black, White — each with primary, tint and shade variants.

### Contrast
Meet WCAG 2.2 level AA, success criterion 1.4.3: text and interactive elements need sufficient contrast (4.5:1 for normal text, 3:1 for large text and UI components). The `govuk-link--inverse` (white-on-dark) style is designed to keep ≥4.5:1. Never use colour as the only means of conveying information (1.4.1).

## Typography

### Typeface — GDS Transport is GOV.UK-only

GDS Transport (also called "New Transport" / `nta`) is the GOV.UK typeface. **Its licence only permits use on services published on GOV.UK** (`*.service.gov.uk` / `gov.uk`). Using it anywhere else — internal tools, council/NHS/ALB sites, or commercial builds — is a font-licensing breach. This is not a style preference; it is a legal constraint (see the "is this service on GOV.UK?" table in `SKILL.md`).

- **On GOV.UK:** use GDS Transport. Serve the font files from your own assets (they ship with GOV.UK Frontend; do not hotlink). Keep the default `$govuk-font-family` and let `$govuk-include-default-font-face` stay `true` so the `@font-face` rules are emitted.
- **Not on GOV.UK:** do **not** ship or reference the GDS Transport files. Override the font to a system stack and suppress the bundled `@font-face`:

  ```scss
  // settings, before @use "govuk-frontend":
  $govuk-include-default-font-face: false;           // don't emit the GDS Transport @font-face
  $govuk-font-family: arial, helvetica, sans-serif;  // or your own brand stack
  ```

  Everything else in the Design System (the type scale, weights, spacing, components) is unchanged — only the typeface differs. Do not deploy the `govuk-frontend` font assets (`/assets/fonts/*`) for an off-GOV.UK build; if they are absent the browser uses the stack above. (A common deployment bug is an SPA catch-all serving `index.html` for missing `/assets/fonts/*`, which then fails to decode as a font — drop the font references entirely instead.)

### Type scale
Sizes are output in relative units (`rem`/`em`) for zoom/magnification. Line heights are multiples of 5px. Sizes are larger on screens ≥ 640px (the "tablet" breakpoint) and smaller below.

| Class | Large screens (≥640px) | Small screens | Typical use |
| --- | --- | --- | --- |
| `govuk-heading-xl` | 48px | 32px | Page `<h1>` (48-point) |
| `govuk-heading-l` | 36px | 27px | Section heading (36-point) |
| `govuk-heading-m` | 24px | 21px | Sub-heading (24-point) |
| `govuk-heading-s` | 19px | 19px | Small heading (19-point) |
| `govuk-body-l` | 24px | 21px | Lead paragraph |
| `govuk-body` / `govuk-body-m` | 19px | 19px | Default body text |
| `govuk-body-s` | 16px | 16px | Small print |

An 80-point size (≈53px on small screens) exists for exceptional use only, and a 27-point size is also "exceptional use". Body copy default is 19px.

### Mixins and overrides
- `govuk-font($size, $weight, $tabular, $line-height)` — full font styling.
- `govuk-font-size($size)` — size only.
- Font-size override classes follow `govuk-!-font-size-19` etc. (point sizes: 80, 48, 36, 27, 24, 19, 16, 14).
- Weight overrides: `govuk-!-font-weight-bold`, `govuk-!-font-weight-regular`.
- Use `govuk-body`, `govuk-heading-*`, `govuk-list`, `govuk-list--bullet`, `govuk-list--number` for paragraphs and lists.

## Layout and grid

The grid is a 12-column-equivalent fractional system inside a fixed-max-width container.

### Container
- `govuk-width-container` — centres content and sets the max page width (default **1020px**), with responsive side gutters. No vertical margin/padding of its own. Can be widened if content needs it (`$govuk-page-width` Sass setting).
- `govuk-main-wrapper` — apply to `<main>`; adds responsive top/bottom padding. Variants: `govuk-main-wrapper--auto-spacing` and `govuk-main-wrapper--l` for extra top spacing when there's no Back link/breadcrumbs above.

### Rows and columns
- `govuk-grid-row` — wraps a row of columns (clearfix).
- Column width classes:
  - `govuk-grid-column-full` (100%)
  - `govuk-grid-column-three-quarters` (75%)
  - `govuk-grid-column-two-thirds` (66.66%) — the standard width for reading content/forms
  - `govuk-grid-column-one-half` (50%)
  - `govuk-grid-column-one-third` (33.33%)
  - `govuk-grid-column-one-quarter` (25%)
- Breakpoint-scoped variants exist, e.g. `govuk-grid-column-two-thirds-from-desktop` (full width below desktop, two-thirds at desktop and up). Combine with a base class for tablet/desktop control.

### Width override (outside the grid)
`govuk-!-width-*` (e.g. `govuk-!-width-two-thirds`, `govuk-!-width-one-half`) constrain individual elements like inputs. Inputs also have fixed-character widths: `govuk-input--width-2`, `-3`, `-4`, `-5`, `-10`, `-20`.

Most service pages put primary content in a single `govuk-grid-column-two-thirds` for an optimal reading measure.

## Spacing

A single responsive spacing scale of points 0–9. Points 0–3 are constant; points 4–9 grow on screens ≥640px.

| Point | Small screens | Large screens (≥640px) |
| --- | --- | --- |
| 0 | 0 | 0 |
| 1 | 5px | 5px |
| 2 | 10px | 10px |
| 3 | 15px | 15px |
| 4 | 15px | 20px |
| 5 | 15px | 25px |
| 6 | 20px | 30px |
| 7 | 25px | 40px |
| 8 | 30px | 50px |
| 9 | 40px | 60px |

(Points 1–3 and the large-screen column match the static scale; small-screen values for 4–9 are smaller. Treat the exact small-screen figures as indicative — verify against source.)

### Applying spacing
- Responsive Sass mixins: `@include govuk-responsive-margin(6, "bottom")`, `@include govuk-responsive-padding(6)`.
- Static spacing function: `padding-top: govuk-spacing(6);` (fixed value, large-screen scale). Negative units allowed: `govuk-spacing(-3)`.
- Override classes (responsive): `govuk-!-margin-bottom-4`, `govuk-!-padding-top-2`, `govuk-!-margin-0`, etc. Pattern: `govuk-!-[margin|padding]-[top|right|bottom|left]-[0-9]` (omit direction for all sides).
- Static override classes: `govuk-!-static-padding-right-5` etc. (same value at every breakpoint).

### Section break
`govuk-section-break` with modifiers `--visible`, `--m`, `--l`, `--xl` to add a horizontal rule / spacing between sections.

## Page template structure

The page template provides the outer HTML skeleton. Top to bottom:
1. `<!DOCTYPE html>` with `<html lang="en" class="govuk-template">`.
2. `<head>`: charset, viewport, `<title>`, theme-color, links to the compiled CSS, and favicon/touch-icon assets shipped with Frontend.
3. `<body class="govuk-template__body">` with a tiny inline script that swaps a `govuk-frontend-supported`/`js-enabled` class onto `<html>` so CSS/JS enhancements only apply when JS is available.
4. Skip link (`govukSkipLink`) → GOV.UK header (`govukHeader`) → optional Service navigation / Phase banner.
5. `<div class="govuk-width-container">` containing Back link/breadcrumbs and `<main class="govuk-main-wrapper" id="main-content" role="main">` with the page content (usually inside a `govuk-grid-row` + `govuk-grid-column-two-thirds`).
6. GOV.UK footer (`govukFooter`).
7. Script that runs `initAll()` to initialise JS components.

(A concrete minimal template is in frontend-conventions.md.)

## Links and focus states

### Link classes
- `govuk-link` — blue, underlined (default link style).
- `govuk-link--no-visited-state` — don't distinguish visited (use for frequently-changing content like admin dashboards).
- `govuk-link--inverse` — white links on dark backgrounds (keeps ≥4.5:1 contrast).
- `govuk-link--no-underline` — remove underline; only where context already signals it's a link (e.g. navigation).
- `govuk-link--muted` — secondary/muted link colour.

Make external links and "opens in new tab" explicit in the link text itself.

### Focus state
GOV.UK's focus style is a solid **yellow** (`#ffdd00`) highlight with a **black bottom border/underline** and black text (`#0b0c0c`) — applied consistently to links, buttons and form controls so keyboard focus is always visible. Never remove focus outlines (`outline: none`) without an equally visible replacement. The focus style is part of meeting WCAG focus-visible requirements; rely on the components' built-in focus styling rather than rolling your own.

## Images
Guidance covers sizing/placement of images within GOV.UK pages; keep images accessible (meaningful `alt` text, decorative images empty `alt=""`), responsive, and don't embed text in images. Verify specifics against the live styles/images page.
