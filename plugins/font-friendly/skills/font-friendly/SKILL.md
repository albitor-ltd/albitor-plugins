---
name: font-friendly
description: "The Friendly font look: Nunito for headings, Nunito Sans for body, data and labels, self-hosted. How the app serves it from its own origin and sets the four --font-* properties. Use when the app's typeface is Friendly."
capability: typeface
---

# Friendly font look

Nunito (rounded terminals) for headings, paired with its plainer sibling Nunito Sans for body, data and labels: a warm, relaxed look. Every family is licensed under the SIL Open Font Licence 1.1.

The plugin seeds these into the app, once. They are the app's own from then on (a later change of
look takes effect through a rebuild, not a re-render):

| Seeded path | What it is |
| --- | --- |
| `web/static/fonts/nunito/Nunito-wght.woff2` | Nunito: the upstream variable font (200 1000), unmodified |
| `web/static/fonts/nunito/OFL.txt` | The upstream licence text for Nunito, with the authors' copyright line |
| `web/static/fonts/nunito-sans/NunitoSans-VF.woff2` | Nunito Sans: the upstream variable font (200 1000), unmodified |
| `web/static/fonts/nunito-sans/OFL.txt` | The upstream licence text for Nunito Sans, with the authors' copyright line |
| `web/static/fonts/fonts.css` | The `@font-face` rules, a relative `src`, `font-display: swap`, and size-adjusted fallbacks |

SvelteKit serves `web/static/` at the site root, so the fonts are at `/fonts/...` on the app's own
origin. Nothing is fetched from another host: the page's policy is `font-src 'self'`, and a font
that does not load is fixed by correcting a path, never by adding a host.

## Apply it

1. Link the stylesheet and preload the body face's file from `web/src/app.html`'s `<head>`
   (see `examples/app.html-head.html`). Preload only that one file.
2. Set the four properties once, unlayered, in `web/src/lib/app/app.css` (see
   `examples/app.css`), then use them: `body { font-family: var(--font-body); }`, headings
   `var(--font-heading)`, tables and figures `var(--font-data)`, buttons, field labels and badges
   `var(--font-label)`. A themeable kit reads the same four properties.
3. Keep the values identical to `preview-tokens.json`: the Look step paints the preview from that
   file, so a different stack in the app would make the build disagree with what was chosen.

## Rules

- Do not rename, subset, convert or re-encode the woff2. It is the distributor's own file, used
  unmodified; its recorded `sha256` is checked by the self-hosting gate.
- Do not add `@import`, a `<link>` to a font host, a `preconnect`, or a `data:` font. Do not widen
  `font-src`.
- Reserved Font Names: none for both families. Neither upstream `OFL.txt` copyright line declares a Reserved Font Name.
- This look ships no italic. Keep `font-style: italic` to the occasional emphasis; the browser
  synthesises it.
- Keep each `"... Fallback"` entry in the stacks, before `Arial`, so the swap does not move the page.
