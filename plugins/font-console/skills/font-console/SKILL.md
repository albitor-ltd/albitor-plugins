---
name: font-console
description: "The Console font look: JetBrains Mono for headings, data and labels, Inter for body, self-hosted. How the app serves it from its own origin and sets the four --font-* properties. Use when the app's typeface is Console."
capability: typeface
---

# Console font look

JetBrains Mono for headings, data and labels, Inter for body: a terminal character. Long text stays in Inter because a mono face reads slowly in paragraphs. Every family is licensed under the SIL Open Font Licence 1.1.

The plugin seeds these into the app, once. They are the app's own from then on (a later change of
look takes effect through a rebuild, not a re-render):

| Seeded path | What it is |
| --- | --- |
| `web/static/fonts/jetbrains-mono/JetBrainsMono-Regular.woff2` | JetBrains Mono 400, the full upstream woff2, unmodified |
| `web/static/fonts/jetbrains-mono/JetBrainsMono-Medium.woff2` | JetBrains Mono 500, the full upstream woff2, unmodified |
| `web/static/fonts/jetbrains-mono/JetBrainsMono-SemiBold.woff2` | JetBrains Mono 600, the full upstream woff2, unmodified |
| `web/static/fonts/jetbrains-mono/JetBrainsMono-Bold.woff2` | JetBrains Mono 700, the full upstream woff2, unmodified |
| `web/static/fonts/jetbrains-mono/OFL.txt` | The upstream licence text for JetBrains Mono, with the authors' copyright line |
| `web/static/fonts/inter/Inter-Regular.woff2` | Inter 400, the full upstream woff2, unmodified |
| `web/static/fonts/inter/Inter-SemiBold.woff2` | Inter 600, the full upstream woff2, unmodified |
| `web/static/fonts/inter/OFL.txt` | The upstream licence text for Inter, with the authors' copyright line |
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
- Mono is never the body face. JetBrains Mono is for headings, data and labels; body stays Inter.
- Only the weights seeded are available: use those named in `examples/app.css`; others are
  synthesised by the browser.
- This look ships no italic. Keep `font-style: italic` to the occasional emphasis.
- Latin text only for now; other scripts fall back to the system stack.
- Keep each family's size-adjusted fallback entry in the stacks, before the generic family, so the
  swap does not move the page:
  - JetBrains Mono: `"JetBrains Mono Fallback"`
  - Inter: `"Inter Fallback"`
