---
name: font-technical
description: "The Technical font look: IBM Plex Sans for headings and body, IBM Plex Mono for data and labels, self-hosted. How the app serves it from its own origin and sets the four --font-* properties. Use when the app's typeface is Technical."
capability: typeface
---

# Technical font look

IBM Plex Sans for headings and body, IBM Plex Mono for data and labels: precise and data-ready. It is the Carbon default. Every family is licensed under the SIL Open Font Licence 1.1.

The plugin seeds these into the app, once. They are the app's own from then on (a later change of
look takes effect through a rebuild, not a re-render):

| Seeded path | What it is |
| --- | --- |
| `web/static/fonts/ibm-plex-sans/IBMPlexSans-Regular.woff2` | IBM Plex Sans 400, the full upstream woff2, unmodified |
| `web/static/fonts/ibm-plex-sans/IBMPlexSans-SemiBold.woff2` | IBM Plex Sans 600, the full upstream woff2, unmodified |
| `web/static/fonts/ibm-plex-sans/OFL.txt` | The upstream licence text for IBM Plex Sans, with the authors' copyright line |
| `web/static/fonts/ibm-plex-mono/IBMPlexMono-Regular.woff2` | IBM Plex Mono 400, the full upstream woff2, unmodified |
| `web/static/fonts/ibm-plex-mono/IBMPlexMono-Medium.woff2` | IBM Plex Mono 500, the full upstream woff2, unmodified |
| `web/static/fonts/ibm-plex-mono/OFL.txt` | The upstream licence text for IBM Plex Mono, with the authors' copyright line |
| `web/static/fonts/fonts.css` | The `@font-face` rules, relative `src`, `font-display: swap`, and size-adjusted fallbacks |

SvelteKit serves `web/static/` at the site root, so the fonts are at `/fonts/...` on the app's own
origin. Nothing is fetched from another host: the page's policy is `font-src 'self'`, and a font
that does not load is fixed by correcting a path, never by adding a host.

## Apply it

1. Link the stylesheet and preload the body face's regular weight from `web/src/app.html`'s
   `<head>` (see `examples/app.html-head.html`). Preload only that one file.
2. Set the four properties once, unlayered, in `web/src/lib/app/app.css` (see
   `examples/app.css`), then use them: `body { font-family: var(--font-body); }`, headings
   `var(--font-heading)`, tables and figures `var(--font-data)`, buttons, field labels and badges
   `var(--font-label)`. A themeable kit reads the same four properties.
3. Keep the values identical to `preview-tokens.json`: the Look step paints the preview from that
   file, so a different stack in the app would make the build disagree with what was chosen.

## Rules

- Do not rename, subset, convert or re-encode any woff2. They are the distributors' own files, used
  unmodified; each recorded `sha256` is checked by the self-hosting gate.
- Do not add `@import`, a `<link>` to a font host, a `preconnect`, or a `data:` font. Do not widen
  `font-src`.
- IBM Plex Sans declares the Reserved Font Name "Plex". Never ship a modified copy under that name.
- IBM Plex Mono declares the Reserved Font Name "Plex". Never ship a modified copy under that name.
- Mono is never the body face. Plex Mono is for data and labels only.
- Only the weights seeded are available: use those named in `examples/app.css`; others are
  synthesised by the browser.
- This look ships no italic. Keep `font-style: italic` to the occasional emphasis.
- Latin text only for now; other scripts fall back to the system stack.
- Keep each family's size-adjusted fallback entry in the stacks, before the generic family, so the
  swap does not move the page:
  - IBM Plex Sans: `"IBM Plex Sans Fallback"`
  - IBM Plex Mono: `"IBM Plex Mono Fallback"`
