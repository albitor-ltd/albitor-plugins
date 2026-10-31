# kit-shadcn

The `component-kit@1` and `component-kit@2` plugin for Svelte
([ADR-0090](https://github.com/albitor-ltd/albitor) sections 2 and 13, albitor-ltd/albitor#3166): a
rewrite of the React-oriented `shadcn` plugin as a Svelte-first kit. It composes into a new app's
`web/` beside `web-svelte`, whose root layout renders the kit's `AppShell`.

## What it provides

- The component-kit@2 standard shell exports from `index.ts`: `AppShell` (now with an optional
  `footer` snippet), `Footer` (new: links and text in a `contentinfo` landmark) and `SignInPage`
  (its own file, `SignInPage.svelte`, also re-exported from `pages/`). The contract is in the repository README; `contract.test.ts` and
  `ContractProbe.svelte` (seeded once) audit it.
- The styled shell pieces in the shadcn style: `AppShell` (skip link, labelled nav, current page,
  one `main`), `Button`, `Field` (label, hint and error tied to the control), `Table`, `Dialog`
  (on [Bits UI](https://bits-ui.com)), `Toaster` and `toast`, `EmptyState`.
- One token file, `tokens.css`: colour roles, radius and type, in light and dark. The `albitor-kit`
  cascade layer is the theming contract: every token is declared inside
  `@layer albitor-kit { :root { ... } }`, so an unlayered declaration of the same custom property
  (an app, or a visual-language plugin) wins. The kit states `"themeable": true` in `plugin.json`,
  which is what lets a visual language compose over it. The platform reads `preview-tokens.json`
  for the build preview.
- Colour contrast that passes out of the box: `contrast.test.ts` computes WCAG 2.2 AA ratios for
  every foreground and background pair the components use, in both schemes, from the shipped
  tokens (axe cannot judge contrast without layout).
- Tests that ship with the app (seeded once): `kit.test.ts` renders the shell pieces on the server and
  runs axe over them, plus behaviour checks (current page, label wiring, live regions, text not
  markup); `dialog.test.ts` mounts the dialog in jsdom for axe, focus and Escape.
- The page templates, `web/src/lib/kit/pages/` (`example`): `ListPage`, `DetailPage`, `FormPage`,
  `DashboardPage`, `SettingsPage` (with `SettingsSection`) and `SignInPage`, over `PageHeader` and
  `StateNotice`. Each is a whole page body built from the kit pieces, with loading, failed and
  empty states, one `<h1>`, and form errors in a focused summary. `pages/PagesShowcase.svelte` and
  `pages/pages.test.ts`  render every template in every state and run axe over them
  (WCAG 2.2 AA), plus behaviour checks. They are the "start from a template" half of the kit
  skill, so a build does not rebuild the same list, form or settings page.
- The `kit-shadcn` skill: rules, components, patterns, tokens and design-system notes (the Bits
  UI primitive for each shadcn component, variants, focus, contrast traps), Svelte-first. It
  declares `metadata.capability: design-system` and the plugin carries the `design-system`
  keyword, so the platform lists the kit as a design system and can select its skill on that
  axis. It also carries `references/accessibility-statement.md` (statement and VPAT/ACR template,
  EAA/ADA/508 guidance); the wider WCAG 2.2 audit content belongs to `test-playwright`.

## Ownership

The components are `example` files: seeded once, then the app's, which is shadcn's model. The
tokens, the contrast test and `web/vitest.config.ts` are seeded once like the rest (the contrast test, `kit.test.ts`, `dialog.test.ts` and the token file stay in the manifest as `managed` only because the platform's kit certification still requires it). `contract.test.ts`, `signin.test.ts` and their probes (`ContractProbe.svelte`, `SignInHarness.svelte`) are seeded once and import the kit's `index.ts` (the `component-kit@2` exports), which is the app's: an app that removes an export fails them. The vitest config extends
web-svelte's `vite.config.ts` so Bits UI's `.svelte` files compile and `mount` resolves to the
client build.

## What it needs

- `web-framework@1` (`web-svelte`). Every file is conditional on it (`when`), and the layout it
  renders imports `web/src/lib/kit/AppShell.svelte` taking `nav`, `path`, `title` (the app's
  name) and `children`.

## Known gaps

- **Dependencies.** The composer records `bits-ui` (and `axe-core`, `jsdom`, `@types/jsdom`,
  `@internationalized/date` for the tests) but does not write them into the app-owned
  `web/package.json`. `scripts/compose-check.sh` installs the declared set before it builds; an
  app composed by the platform needs them added until the composer does it.
- **No Tailwind.** shadcn-svelte's own components use Tailwind and `tailwind-variants`; web-svelte's
  `vite.config.ts` has no Tailwind plugin, so this kit styles with scoped CSS over the
  tokens and uses Bits UI for behaviour. Adding Tailwind is a web-svelte change.
- **Colour contrast in a real browser** is audited from the tokens, not from rendered pixels.
  The product e2e suite (`test-playwright`) is where a rendered audit belongs.

## Layout

```text
.claude-plugin/plugin.json   schema-2 manifest
preview-tokens.json          the preview palette (platform reads it; equals tokens.css light)
templates/web/**             the files composed into the app, at their repository paths
skills/kit-shadcn/           the skill and its references
```

The legacy `shadcn` plugin (schema 1, React) stays in the catalogue until it is retired separately.
