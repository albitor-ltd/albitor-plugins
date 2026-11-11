---
name: font-geometric
description: "The Geometric font look: Plus Jakarta Sans for headings, body, data and labels, self-hosted. How the app serves it from its own origin and sets the four --font-* properties. Use when the app's typeface is Geometric."
capability: typeface
---

# Geometric font look

One family, Plus Jakarta Sans, for every role: a modern geometric sans with round, open forms. The
family is licensed under the SIL Open Font Licence 1.1.

The plugin seeds three things into the app, once. They are the app's own from then on (a later
change of look takes effect through a rebuild, not a re-render):

| Seeded path | What it is |
| --- | --- |
| `web/static/fonts/plus-jakarta-sans/PlusJakartaSans-Variable.woff2` | The full upstream variable font, weights 200 to 800, unmodified |
| `web/static/fonts/plus-jakarta-sans/OFL.txt` | The upstream licence text, with the authors' copyright line |
| `web/static/fonts/fonts.css` | The `@font-face` rules, a relative `src`, `font-display: swap`, and a size-adjusted fallback |

SvelteKit serves `web/static/` at the site root, so the font is at `/fonts/plus-jakarta-sans/...` on
the app's own origin. Nothing is fetched from another host: the page's policy is `font-src 'self'`,
and a font that does not load is fixed by correcting a path, never by adding a host.

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
- Plus Jakarta Sans declares no Reserved Font Name, but keep the file unmodified all the same.
- Weights run 200 to 800; the look uses 400, 600 and 700. Tabular figures are not the default, so
  set `font-variant-numeric: tabular-nums` on tables and figures.
- This look ships no italic. Keep `font-style: italic` to the occasional emphasis; the browser
  synthesises it.
- Latin text only for now; other scripts fall back to the system stack.
- The `@font-face` fallback is sized against Arial at the regular weight; keep the
  `"Plus Jakarta Sans Fallback"` entry in each stack, before `Arial`, so the swap does not move the
  page.
