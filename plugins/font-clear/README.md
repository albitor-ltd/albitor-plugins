# Font: Clear (`font-clear`)

The Clear font look (ADR-0107): Source Sans 3 for headings, body, data and labels. A `typeface@1`
plugin and the worked example for the other looks.

- **Family.** Source Sans 3 `3.052R` by Adobe, the variable upright font (weights 200 to 900),
  the distributor's own `SourceSans3VF-Upright.otf.woff2` from
  <https://github.com/adobe-fonts/source-sans/releases/tag/3.052R>. Unmodified and unsubset.
- **Licence.** SIL Open Font Licence 1.1; the upstream text is `OFL.txt` beside the font. Reserved
  Font Name: `Source` (from the upstream copyright line), recorded as `rfn` in the manifest.
- **Seeds** (once, app-owned): the font, `OFL.txt` and `fonts.css` under `web/static/fonts/`.
- **`preview-tokens.json`** holds the four stacks (`font_heading`, `font_body`, `font_data`,
  `font_label`) the Look step paints.
- **Skill.** `skills/font-clear/SKILL.md` and its examples: linking `fonts.css`, preloading the body
  file and setting the four `--font-*` properties.

`scripts/validate-fonts.sh` (run by `just ci`) checks the licence, the OFL text, the woff2 signature
and hash, the same-origin CSS and the recorded RFN.
