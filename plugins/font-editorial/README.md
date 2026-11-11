# Font: Editorial (`font-editorial`)

The Editorial font look (ADR-0107): Source Serif 4 for headings and Source Sans 3 for body, data and
labels. A `typeface@1` plugin.

- **Source Serif 4** `4.005R` by Adobe: the static Semibold (600) and Bold (700) files,
  `SourceSerif4-Semibold.ttf.woff2` and `SourceSerif4-Bold.ttf.woff2` from
  <https://github.com/adobe-fonts/source-serif/releases/tag/4.005R>. Unmodified and unsubset. The
  upstream variable file is 427 KB, so the two static weights the look uses are shipped instead.
- **Source Sans 3** `3.052R` by Adobe: the same variable upright file as `font-clear`
  (`SourceSans3VF-Upright.otf.woff2`, weights 200 to 900), in this plugin's own directory with its
  own `OFL.txt`.
- **Licence.** SIL Open Font Licence 1.1 for both; the upstream texts are the `OFL.txt` beside each
  family. Both copyright lines declare the Reserved Font Name `Source`, recorded as `rfn`.
- **Seeds** (once, app-owned): the fonts, both `OFL.txt` files and `fonts.css` under
  `web/static/fonts/`.
- **`preview-tokens.json`** holds the four stacks (`font_heading`, `font_body`, `font_data`,
  `font_label`) the Look step paints.
- **Skill.** `skills/font-editorial/SKILL.md` and its examples: linking `fonts.css`, preloading the
  body file and setting the four `--font-*` properties.
