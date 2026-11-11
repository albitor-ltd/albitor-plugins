# Font: Geometric (`font-geometric`)

The Geometric font look (ADR-0107): Plus Jakarta Sans for headings, body, data and labels. A
`typeface@1` plugin.

- **Family.** Plus Jakarta Sans `2.071` (release `2.7.1`) by the Plus Jakarta Sans Project Authors,
  the variable upright font (weights 200 to 800), the upstream repository's own
  `fonts/webfonts/PlusJakartaSans[wght].woff2` at commit `6d4ffaf` (the 2.7.1 release zip carries
  no woff2). Unmodified and unsubset; saved as `PlusJakartaSans-Variable.woff2`.
- **Licence.** SIL Open Font Licence 1.1; the upstream text is `OFL.txt` beside the font. The
  copyright line declares no Reserved Font Name, so `rfn` is `none`.
- **Seeds** (once, app-owned): the font, `OFL.txt` and `fonts.css` under `web/static/fonts/`.
- **`preview-tokens.json`** holds the four stacks (`font_heading`, `font_body`, `font_data`,
  `font_label`) the Look step paints.
- **Skill.** `skills/font-geometric/SKILL.md` and its examples: linking `fonts.css`, preloading the
  body file and setting the four `--font-*` properties.
