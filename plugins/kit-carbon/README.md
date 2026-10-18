# kit-carbon

The `component-kit@1` plugin for Svelte in the IBM Carbon style
([ADR-0090](https://github.com/albitor-ltd/albitor) sections 2 and 13, albitor-ltd/albitor#3167): a
rewrite of the React and web-component `carbon` plugin as a Svelte-first kit on
[carbon-components-svelte](https://svelte.carbondesignsystem.com) and
[carbon-icons-svelte](https://github.com/carbon-design-system/carbon-icons-svelte). It composes into
a new app's `web/` beside `web-svelte`, whose root layout renders the kit's `AppShell`. It has the
same shell pieces, names and props as `kit-shadcn`, so a build can swap one kit for the other.

## What it provides

- The styled shell pieces in the Carbon style: `AppShell` (Carbon's UI shell: header, skip link,
  a labelled side nav with the current page marked, one `main`), `Button`, `Field` (label, hint and
  error tied to the control), `Table`, `Dialog` (on Carbon's `Modal`), `Toaster` and `toast`,
  `EmptyState`.
- Carbon's own themes: the page wears `white` in light and `g100` (Gray 100) in dark, following the
  visitor's preference unless `<html data-theme="light">` or `"dark"` forces one (`theme.ts`).
- Colour contrast that passes out of the box: `contrast.test.ts` reads the installed Carbon
  stylesheet and computes WCAG 2.2 AA ratios for every foreground and background pair the
  components use, in both themes (axe cannot judge contrast without layout).
- Tests that ship with the app (managed): `kit.test.ts` renders the shell pieces on the server and
  runs axe over them, plus behaviour checks (current page, label wiring, live regions, button types
  and variants, text not markup); `dialog.test.ts` mounts the dialog in jsdom for axe, focus (trap,
  Escape, return) and the Close button; `shell.test.ts` mounts the shell and checks the theme
  follows the colour-scheme preference and `data-theme`, including when either changes.
- The page templates, `web/src/lib/kit/pages/` (`example`): `ListPage`, `DetailPage`, `FormPage`,
  `DashboardPage`, `SettingsPage` (with `SettingsSection`) and `SignInPage`, over `PageHeader` and
  `StateNotice`. Each is a whole page body built from the kit pieces, with loading, failed and
  empty states, one `<h1>`, and form errors in a focused summary. `pages/PagesShowcase.svelte` and
  `pages/pages.test.ts` (`managed`) render every template in every state and run axe over them
  (WCAG 2.2 AA), plus behaviour checks. They are the "start from a template" half of the kit
  skill, so a build does not rebuild the same list, form or settings page.
- The `kit-carbon` skill: rules, components, patterns and tokens, Svelte-first. The shared WCAG 2.2
  content belongs to `test-playwright`, so it is not carried here.

## Not themeable

Carbon is not themeable by a visual-language plugin. Its look is its themes, and the kit adds no
token layer: it ships no `tokens.css` and no `albitor-kit` cascade layer (the convention
`kit-shadcn` uses to say "an unlayered override wins"). A visual-language pack that restyles type,
radius, density or colour would fight Carbon's own stylesheet, so do not pair the two.

`component-kit@1` has no manifest field to declare this yet, so the platform's resolver cannot refuse
the pairing from the kit's side (albitor-ltd/albitor#3236 asks for one). Until then the declaration
is this section, the skill, and a test (`TestKitCarbonIsNotThemeable`) that fails if the kit ever
ships the layer. A policy pack that forbids `visual-language` (as the GOV.UK pack does) refuses the
pairing today.

## Ownership

The components and `theme.ts` are `example` files: seeded once, then the app's. The contrast test,
the Showcase, the three test files and `web/vitest.config.ts` are `managed`. No managed file imports
an `example` file by name except the Showcase (which renders the components); the theme tests check
the `theme` attribute Carbon puts on `<html>` through the mounted shell, so an app may rewrite
`theme.ts` and keep the gate as long as the page still follows the preference and `data-theme`. The vitest config extends
web-svelte's `vite.config.ts` and inlines `carbon-components-svelte` so its `.svelte` files compile,
and resolves `mount` to the client build.

## What it needs

- `web-framework@1` (`web-svelte`). Every file is conditional on it (`when`), and the layout it
  renders imports `web/src/lib/kit/AppShell.svelte` taking `nav`, `path`, `title` (the app's
  name) and `children`.

## Known gaps

- **Dependencies.** The composer records the packages (and `axe-core`, `jsdom`, `@types/jsdom` for
  the tests) but does not write them into the app-owned `web/package.json`.
  `scripts/compose-check.sh` installs the declared set before it builds; an app composed by the
  platform needs them added until the composer does it.
- **IBM telemetry runs at install.** `carbon-components-svelte` depends on `@ibm/telemetry-js`
  (`^1.5.0`, a range, so it floats) and runs it as a `postinstall` (`ibmtelemetry`), which sends
  anonymous usage data to IBM. Install-time code that floats is younger than this repository's
  14-day pin rule would allow (1.12.0 was published 2026-10-08). The manifest therefore also pins
  `@ibm/telemetry-js@1.11.0` (2026-01-24) as a direct dependency, the only pin mechanism a plugin
  has; it holds only once the dependency is written into the app (composer gap, albitor-ltd/albitor#3235),
  and until then the app's lockfile decides. Set `IBM_TELEMETRY_DISABLED=true` for the install:
  `scripts/compose-check.sh` does, and it fails if the pinned version is not the one installed. No
  other environment sets it today. The platform's build and certification environments, and the
  developer's own machine, must, and that is for albitor to decide.
- **Dark first paint.** The theme is chosen when the shell mounts, so a visitor who prefers dark sees
  a white frame for an instant first. Fixing it needs a script in `app.html` (web-svelte's file).
- **Table names its region and its caption**, so a screen reader may say the caption twice. The
  scroll region needs a name to be reachable; naming it by the caption is the least wrong choice.
- **Svelte 4 components.** carbon-components-svelte is written in the Svelte 4 style (slots,
  `on:` events) and runs in a Svelte 5 app through its compatibility layer. The kit's wrappers hide
  that behind runes-style props.
- **Dialog focus return** is Carbon's, done when the closing transition ends. The test simulates
  that event, because jsdom runs no transitions; the real browser behaviour belongs to the product
  e2e suite (`test-playwright`).
- **The modal title is an `h3`** whatever the page's outline above it, so `dialog.test.ts` turns
  off axe's `heading-order` for the dialog only.
- **Colour contrast in a real browser** is audited from the stylesheet, not from rendered pixels.

## Layout

```text
.claude-plugin/plugin.json   schema-2 manifest
preview-tokens.json          the preview palette (platform reads it; equals Carbon white)
templates/web/**             the files composed into the app, at their repository paths
skills/kit-carbon/           the skill and its references
```

The legacy `carbon` plugin (schema 1, React and web components) stays in the catalogue until it is
retired separately.
