# Look: Consumer (`look-consumer`)

Friendly and warm. Rounded type, 24px corners, 48px targets, warm white neutrals with a berry accent, spacious layouts.

A `visual-language@1` plugin: a look pack. It ships `preview-tokens.json` (the token set the
platform paints the preview and derives the light and dark scheme from) and one skill,
`look-consumer`, telling builders how to apply the look to any web app. It contributes two seeded files,
`web/src/lib/look/look.json` (a copy of the tokens for the build to read) and
`web/src/lib/look/EmptyArt.svelte` (the look's art for first-run states, drawn in the app's tokens),
and composes over any themeable component kit.

**Suits.** Friendly, phone-first apps: personal, family, community, wellness, hobby, learning and booking tools used in short sessions, where warmth and ease beat density. Works with any shell and first screen the app picks. Avoid for data-heavy admin and multi-column tables.

See `skills/look-consumer/SKILL.md` for typography, density, shape, elevation, colour, the look's signature,
phone navigation and dark-mode guidance.

Beside the skill:

- `examples/web/src/routes/+layout.svelte`: the root layout wired with the look's `layout`, `phoneNav`
  and `icons` props for `AppShell` (structure and style stay separate axes).
- `examples/web/src/lib/look/pages/`: styled examples of each kit page template (overview, list,
  detail, form, settings, first run, step flow) written against the kit-shadcn components, with a
  test that runs axe over each, and a non-CRUD example app
  (Classes, Class and Bookings: a catalogue and booking example app; albitor-ltd/albitor-plugins#283). The overview is the dashboard first screen only.
- `rubric.md`: a short checklist for the visual check of a built page.

The manifest names this look's default font look, Friendly (`font-friendly`), in
`default_font_look`: a suggestion when the person has not picked one, outranked by their pick and by
any policy lock.
Validate the tokens with `scripts/validate-preview-tokens.sh` from the repository root.

Notes. `-apple-system` is deliberately omitted from the font stacks: the platform's bare-family check refuses it (`system-ui` covers the same face). A look only applies to a component kit that is `themeable: true`.
