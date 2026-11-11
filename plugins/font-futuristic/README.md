# Font: Futuristic (`font-futuristic`)

The Futuristic font look (ADR-0107): Space Grotesk for headings, Inter for body, data and labels: bold, forward-looking headings over neutral text. A `typeface@1` plugin, built on the `font-clear`
example.

- **Space Grotesk** `2.0.0`, weights 300 700. Reserved Font Name: `none`.
- **Inter** `4.1`, weights 400, 600. Reserved Font Name: `none`.

Files are the distributors' own full woff2, unmodified and unsubset; `source` and `sha256` are in
the manifest's `fonts` block and the upstream `OFL.txt` sits beside each family. The seeded files
total about 275 KB of woff2.

- **Seeds** (once, app-owned): the fonts, each family's `OFL.txt` and `fonts.css` under
  `web/static/fonts/`.
- **`preview-tokens.json`** holds the four stacks (`font_heading`, `font_body`, `font_data`,
  `font_label`) the Look step paints.
- **Skill.** `skills/font-futuristic/SKILL.md` and its examples: linking `fonts.css`, preloading the body
  file and setting the four `--font-*` properties.

`scripts/validate-fonts.sh` (run by `just ci`) checks the licence, the OFL text, the woff2 signature
and hash, the same-origin CSS and the recorded RFN.
