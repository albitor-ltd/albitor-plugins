# kit-gds

The `component-kit@1` plugin for Svelte in the GOV.UK Design System style
(albitor-ltd/albitor#3563). It absorbs the `gds` pack: the GOV.UK skills travel with the kit, and
the kit makes the components follow them. It composes into a new app's `web/` beside `web-svelte`,
whose root layout renders the kit's `AppShell`.

## What it provides

- The shell pieces in [GOV.UK Frontend](https://frontend.design-system.service.gov.uk/) v6.4.0
  markup and classes: `AppShell` (skip link, a text-only header, service navigation with the
  current page marked, one `main`), `Button`, `Field` (label, hint and error tied to the control),
  `ErrorSummary` (GOV.UK error summary: focus on mount, each entry links to its field's id),
  `Table` and `Dialog`. GOV.UK has no modal, so `Dialog` is a GOV.UK-styled native `<dialog>`
  whose first focus is the Close button, never a destructive action you pass in.
- `govuk.scss`: the Frontend styles those components wear, compiled from the pinned
  `govuk-frontend` package with `sass`. The service navigation is restated in that file rather than
  imported, because Frontend's stylesheet for it carries an Internet Explorer media-query hack that
  the app's CSS minifier refuses.
- A base layer in `govuk.scss`: bare `body`, `h1`-`h3`, `p` and `a` (link, hover, visited and
  focus states) render as GOV.UK with no `govuk-*` class, so unstyled pages such as auth-cognito's
  sign-in render correctly. It is wrapped in `:where()`, so it has no specificity and every class
  still wins.
- **No Crown, no GOV.UK logotype, no GDS Transport font** (the font falls back to arial). They are
  licensed for services on GOV.UK only; this kit is for the apps that are not. The header is text.
- Tests that ship with the app (managed): `kit.test.ts` renders the shell pieces on the server and
  runs axe over them, plus behaviour checks (current page, label wiring, text not markup, a
  text-only header); `dialog.test.ts` mounts the dialog in jsdom for axe, its open state and its initial focus;
  `contrast.test.ts` reads GOV.UK Frontend's colours from the installed stylesheet and checks every
  pair the components use against WCAG 2.2 AA.
- The `govuk-design-system` and `govuk-accessibility` skills from `gds` (components, patterns,
  styles; WCAG 2.2 AA, the public sector regulations, testing, the accessibility statement, and the
  audit and statement procedures), each tagged `metadata.capability: design-system`.
- `preview-tokens.json`: GOV.UK Frontend 6.4.0's palette for the build preview.

## Root element classes

The kit fills web-svelte's `web.root-classes` slot (albitor#3707) with GOV.UK Frontend's page
template classes: `govuk-template` on `<html>` and `govuk-template__body` on `<body>`. These need
web-svelte 0.4.0 or later, which hosts the slot in `app.html`.

Frontend's template also runs an inline script first in `<body>` that adds `js-enabled` and
`govuk-frontend-supported` (when `noModule` is supported) to the body. This is a client-rendered
SPA, so `AppShell` does it instead (in its script body, before children mount) and removes only what it added on destroy. No
inline script means no CSP `script-src` exception; the classes gate only JS-enhanced styling, so
a page without JS is the plain one. `rootclasses.test.ts` (managed) covers it. The classes are set while `AppShell` initialises, before any child mounts, so a component that calls `initAll()` in its own `onMount` already sees them.

## Brand colour

The one token an app sets, in `govuk.scss` (`$govuk-brand-colour`, marked at the top of the file):

- **On GOV.UK** (a `service.gov.uk` or `gov.uk` domain): leave it at `#1d70b8`.
- **Off GOV.UK**: set it to the app's own brand colour, and change nothing else. The header band
  follows it (`--govuk-brand-colour`). The header text is picked in the same file, white or
  near-black, to keep 4.5:1 on whatever you set, and `contrast.test.ts` checks that pair from the
  compiled stylesheet.

In Frontend 6 `$govuk-brand-colour` is a deprecated alias that cannot be configured, so the file
sets it where Frontend reads it, the `brand` entry of `$govuk-functional-colours`. This is what the
platform's brand-colour check looks for: unchanged on GOV.UK, set off it.

## Not themeable

`"themeable": false`: GOV.UK's look is not overridden by a visual-language plugin, so a
`visual-language@1` plugin composed with this kit is refused as `kit_not_themeable`. The platform
recognises the kit as GOV.UK by its name and keywords (`core.IsGOVUKDesignSystem`).

## Ownership

The components and `govuk.scss` are `example` files: seeded once, then the app's. The contrast and
axe tests, `Showcase.svelte` and `web/vitest.config.ts` are `managed`. The `gds` plugin is
unchanged and is retired separately.
