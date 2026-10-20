# kit-gds

The `component-kit@1` plugin for Svelte in the GOV.UK Design System style
(albitor-ltd/albitor#3563). It absorbs the `gds` pack: the GOV.UK skills travel with the kit, and
the kit makes the components follow them. It composes into a new app's `web/` beside `web-svelte`,
whose root layout renders the kit's `AppShell`.

## What it provides

- The shell pieces in [GOV.UK Frontend](https://frontend.design-system.service.gov.uk/) v6.4.0
  markup and classes: `AppShell` (skip link, a text-only header, service navigation with the
  current page marked, one `main`), `Button`, `Field` (label, hint and error tied to the control),
  `Table` and `Dialog`. GOV.UK has no modal, so `Dialog` is a GOV.UK-styled native `<dialog>`.
- `govuk.scss`: the Frontend styles those components wear, compiled from the pinned
  `govuk-frontend` package with `sass`. The service navigation is restated in that file rather than
  imported, because Frontend's stylesheet for it carries an Internet Explorer media-query hack that
  the app's CSS minifier refuses.
- **No Crown, no GOV.UK logotype, no GDS Transport font** (the font falls back to arial). They are
  licensed for services on GOV.UK only; this kit is for the apps that are not. The header is text.
- Tests that ship with the app (managed): `kit.test.ts` renders the shell pieces on the server and
  runs axe over them, plus behaviour checks (current page, label wiring, text not markup, a
  text-only header); `dialog.test.ts` mounts the dialog in jsdom for axe and its open state;
  `contrast.test.ts` reads GOV.UK Frontend's colours from the installed stylesheet and checks every
  pair the components use against WCAG 2.2 AA.
- The `govuk-design-system` and `govuk-accessibility` skills from `gds` (components, patterns,
  styles; WCAG 2.2 AA, the public sector regulations, testing, the accessibility statement, and the
  audit and statement procedures), each tagged `metadata.capability: design-system`.
- `preview-tokens.json`: GOV.UK Frontend 6.4.0's palette for the build preview.

## Not themeable

`"themeable": false`: GOV.UK's look is not overridden by a visual-language plugin, so a
`visual-language@1` plugin composed with this kit is refused as `kit_not_themeable`. The platform
recognises the kit as GOV.UK by its name and keywords (`core.IsGOVUKDesignSystem`).

## Ownership

The components and `govuk.scss` are `example` files: seeded once, then the app's. The contrast and
axe tests, `Showcase.svelte` and `web/vitest.config.ts` are `managed`. The `gds` plugin is
unchanged and is retired separately.
