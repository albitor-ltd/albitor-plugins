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
  runs axe over them, plus behaviour checks (current page, label wiring, live regions, text not
  markup); `dialog.test.ts` mounts the dialog in jsdom for axe, focus (trap, Escape, return) and
  the Close button.
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
the Showcase, the two test files and `web/vitest.config.ts` are `managed`. The vitest config extends
web-svelte's `vite.config.ts` and inlines `carbon-components-svelte` so its `.svelte` files compile,
and resolves `mount` to the client build.

## What it needs

- `web-framework@1` (`web-svelte`). Every file is conditional on it (`when`), and the layout it
  renders imports `web/src/lib/kit/AppShell.svelte` taking `nav`, `path` and `children`.

## Known gaps

- **Dependencies.** The composer records the packages (and `axe-core`, `jsdom`, `@types/jsdom` for
  the tests) but does not write them into the app-owned `web/package.json`.
  `scripts/compose-check.sh` installs the declared set before it builds; an app composed by the
  platform needs them added until the composer does it.
- **IBM telemetry.** `carbon-components-svelte` declares a `postinstall` (`ibmtelemetry`, from
  `@ibm/telemetry-js`) that sends anonymous usage data to IBM. Set `IBM_TELEMETRY_DISABLED=true` for
  the install; `scripts/compose-check.sh` does. Whether the platform's build environment should set
  it is for albitor to decide.
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
