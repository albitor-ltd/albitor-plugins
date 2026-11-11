---
name: font-editorial
description: "The Editorial font look: Source Serif 4 headings over Source Sans 3 body, data and labels, self-hosted. How the app serves it from its own origin and sets the four --font-* properties. Use when the app's typeface is Editorial."
capability: typeface
---

# Editorial font look

A reading-led pairing: Source Serif 4 (Adobe) for headings, Source Sans 3 (Adobe) for body, data and
labels. Both families are licensed under the SIL Open Font Licence 1.1.

The plugin seeds these into the app, once. They are the app's own from then on (a later change of
look takes effect through a rebuild, not a re-render):

| Seeded path | What it is |
| --- | --- |
| `web/static/fonts/source-serif-4/SourceSerif4-Semibold.ttf.woff2` | Source Serif 4, weight 600, unmodified |
| `web/static/fonts/source-serif-4/SourceSerif4-Bold.ttf.woff2` | Source Serif 4, weight 700, unmodified |
| `web/static/fonts/source-serif-4/OFL.txt` | The upstream licence text, with the authors' copyright line |
| `web/static/fonts/source-sans-3/SourceSans3VF-Upright.otf.woff2` | Source Sans 3 variable, weights 200 to 900, unmodified |
| `web/static/fonts/source-sans-3/OFL.txt` | The upstream licence text, with the authors' copyright line |
| `web/static/fonts/fonts.css` | The `@font-face` rules, relative `src`, `font-display: swap`, and size-adjusted fallbacks |

SvelteKit serves `web/static/` at the site root, so the fonts are at `/fonts/...` on the app's own
origin. Nothing is fetched from another host: the page's policy is `font-src 'self'`, and a font
that does not load is fixed by correcting a path, never by adding a host.

## Apply it

1. Link the stylesheet and preload the body face's file from `web/src/app.html`'s `<head>`
   (see `examples/app.html-head.html`). Preload only that one file, the Source Sans 3 body.
2. Set the four properties once, unlayered, in `web/src/lib/app/app.css` (see
   `examples/app.css`), then use them: `body { font-family: var(--font-body); }`, headings
   `var(--font-heading)`, tables and figures `var(--font-data)`, buttons, field labels and badges
   `var(--font-label)`. A themeable kit reads the same four properties.
3. Keep the values identical to `preview-tokens.json`: the Look step paints the preview from that
   file, so a different stack in the app would make the build disagree with what was chosen.

## Rules

- Do not rename, subset, convert or re-encode any woff2. They are the distributor's own files, used
  unmodified; their recorded `sha256` is checked by the self-hosting gate.
- Do not add `@import`, a `<link>` to a font host, a `preconnect`, or a `data:` font. Do not widen
  `font-src`.
- Both families declare the Reserved Font Name "Source". Never ship a modified copy under that name.
- Source Serif 4 is for headings only, at 600 or 700. Do not use it for body text, and do not ask
  for other weights: only those two are shipped, and a missing weight is synthesised badly.
- This look ships no italic. Keep `font-style: italic` to the occasional emphasis; the browser
  synthesises it.
- Latin text only for now; other scripts fall back to the system stack.
- The `@font-face` fallbacks are size-adjusted; keep `"Source Serif 4 Fallback"` and
  `"Source Sans 3 Fallback"` in their stacks, before the system face, so the swap does not move the
  page.
