# Look: Consumer (`look-consumer`)

Friendly and warm. Rounded type, 24px corners, 48px targets, cream neutrals with a berry accent, spacious layouts.

A `visual-language@1` plugin: a look pack. It ships `preview-tokens.json` (the token set the
platform paints the preview and derives the light and dark scheme from) and one skill,
`look-consumer`, telling builders how to apply the look to any web app. It contributes one seeded file,
`web/src/lib/look/look.json` (a copy of the tokens for the build to read), and composes over any
themeable component kit.

**Suits.** Mobile-first personal and lifestyle apps: habit, planning, family, wellness and hobby tools used in short sessions on a phone, where warmth and ease beat density. Avoid for data-heavy admin and multi-column tables.

See `skills/look-consumer/SKILL.md` for typography, density, shape, elevation, colour and dark-mode guidance.
Validate the tokens with `scripts/validate-preview-tokens.sh` from the repository root.
