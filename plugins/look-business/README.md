# Look: Business (`look-business`)

Dense, exact and quiet. Navy neutrals, a single blue accent, 2px corners, compact rows, tabular figures, no shadows.

A `visual-language@1` plugin: a look pack. It ships `preview-tokens.json` (the token set the
platform paints the preview and derives the light and dark scheme from) and one skill,
`look-business`, telling builders how to apply the look to any web app. It contributes one seeded file,
`web/src/lib/look/look.json` (a copy of the tokens for the build to read), and composes over any
themeable component kit.

**Suits.** Enterprise and line-of-business apps: admin consoles, finance, back office and operations tools used by staff for long sessions, where comparing exact numbers across many rows matters more than first impressions. Avoid for mobile-first consumer or marketing surfaces.

See `skills/look-business/SKILL.md` for typography, density, shape, elevation, colour and dark-mode guidance.
Validate the tokens with `scripts/validate-preview-tokens.sh` from the repository root.
