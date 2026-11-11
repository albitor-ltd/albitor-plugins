# Font: Friendly (`font-friendly`)

The Friendly font look (ADR-0107): Nunito for headings, Nunito Sans for body, data and labels. A `typeface@1` plugin.

- **Nunito.** Nunito 3.602 (full variable TTF converted losslessly to woff2), 200 1000 variable upright, `Nunito-wght.woff2` from <https://github.com/googlefonts/nunito/tree/8c6a9bb9732545b9ed53f29ec5e1ab0ff53c4e6f>. Unmodified.
  Reserved Font Name: none.
- **Nunito Sans.** Nunito Sans 3.101 (full variable TTF converted losslessly to woff2), 200 1000 variable upright, `NunitoSans-VF.woff2` from <https://github.com/googlefonts/NunitoSans/tree/058bd7a2f33d6ad5ef1df985b3db403622016a8c>. Unmodified.
  Reserved Font Name: none.
- **Licence.** SIL Open Font Licence 1.1; each upstream text is `OFL.txt` beside its font.
- **Seeds** (once, app-owned): the fonts, their `OFL.txt` files and `fonts.css` under
  `web/static/fonts/`.
- **`preview-tokens.json`** holds the four stacks (`font_heading`, `font_body`, `font_data`,
  `font_label`) the Look step paints.
- **Skill.** `skills/font-friendly/SKILL.md` and its examples: linking `fonts.css`, preloading the body
  file and setting the four `--font-*` properties.

Upstream publishes these two families as variable TTF only, so each woff2 is a lossless format
conversion of the full upstream TTF (fontTools 4.68.0 with brotli 1.2.1; no glyph, table or axis
change, not a subset). The TTF URL, commit and sha256 are in each `source`; the manifest `sha256`
is the woff2's.
