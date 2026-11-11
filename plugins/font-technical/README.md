# Font: Technical (`font-technical`)

The Technical font look (ADR-0107): IBM Plex Sans for headings and body, IBM Plex Mono for data and labels: precise and data-ready. It is the Carbon default. A `typeface@1` plugin, built on the `font-clear`
example.

- **IBM Plex Sans** `1.1.0`, weights 400, 600. Reserved Font Name: `Plex`.
- **IBM Plex Mono** `2.5.0`, weights 400, 500. Reserved Font Name: `Plex`.

Files are the distributors' own full woff2, unmodified and unsubset; `source` and `sha256` are in
the manifest's `fonts` block and the upstream `OFL.txt` sits beside each family. The seeded files
total about 230 KB of woff2.

- **Seeds** (once, app-owned): the fonts, each family's `OFL.txt` and `fonts.css` under
  `web/static/fonts/`.
- **`preview-tokens.json`** holds the four stacks (`font_heading`, `font_body`, `font_data`,
  `font_label`) the Look step paints.
- **Skill.** `skills/font-technical/SKILL.md` and its examples: linking `fonts.css`, preloading the body
  file and setting the four `--font-*` properties.

`scripts/validate-fonts.sh` (run by `just ci`) checks the licence, the OFL text, the woff2 signature
and hash, the same-origin CSS and the recorded RFN.
