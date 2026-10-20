# Look: Startup (`look-startup`)

Crisp and quiet. Near-achromatic neutrals, one teal accent, 10px corners, hairlines plus a soft layered shadow, hierarchy from type not decoration.

A `visual-language@1` plugin: a look pack. It ships `preview-tokens.json` (the token set the
platform paints the preview and derives the light and dark scheme from) and one skill,
`look-startup`, telling builders how to apply the look to any web app. It contributes one seeded file,
`web/src/lib/look/look.json` (a copy of the tokens for the build to read), and composes over any
themeable component kit.

**Suits.** SaaS products and developer tools: dashboards, workspaces, settings and onboarding for people who use the product daily and judge it on speed and clarity. Avoid for dense back-office tables (use Business) or mobile-first lifestyle apps (use Consumer).

See `skills/look-startup/SKILL.md` for typography, density, shape, elevation, colour and dark-mode guidance.
Validate the tokens with `scripts/validate-preview-tokens.sh` from the repository root.

Notes. `-apple-system` is deliberately omitted from the font stacks: the platform's bare-family check refuses it (`system-ui` covers the same face). A look only applies to a component kit that is `themeable: true`.
