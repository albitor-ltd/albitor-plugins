# Font: Friendly (`font-friendly`)

The Friendly font look (ADR-0107): Nunito for headings, Nunito Sans for body, data and labels. A `typeface@1` plugin.

- **Nunito.** Nunito 3.602 via @fontsource-variable/nunito 5.3.0, 200 1000 variable upright, `Nunito-latin-wght-normal.woff2` from <https://www.npmjs.com/package/@fontsource-variable/nunito/v/5.3.0>. Unmodified.
  Reserved Font Name: none.
- **Nunito Sans.** Nunito Sans 3.101 via @fontsource-variable/nunito-sans 5.3.0, 200 1000 variable upright, `NunitoSans-latin-wght-normal.woff2` from <https://www.npmjs.com/package/@fontsource-variable/nunito-sans/v/5.3.0>. Unmodified.
  Reserved Font Name: none.
- **Licence.** SIL Open Font Licence 1.1; each upstream text is `OFL.txt` beside its font.
- **Seeds** (once, app-owned): the fonts, their `OFL.txt` files and `fonts.css` under
  `web/static/fonts/`.
- **`preview-tokens.json`** holds the four stacks (`font_heading`, `font_body`, `font_data`,
  `font_label`) the Look step paints.
- **Skill.** `skills/font-friendly/SKILL.md` and its examples: linking `fonts.css`, preloading the body
  file and setting the four `--font-*` properties.
