Source: Material UI (MUI, https://mui.com/), Svelte Material UI (SMUI, https://sveltematerialui.com/), m3-svelte (https://github.com/KTibow/m3-svelte), and Material Design (https://m3.material.io/, https://m2.material.io/). MUI, SMUI, and m3-svelte are MIT-licensed. Distilled reference — verify against the live installation/usage pages for current detail, especially across major versions.

# Consuming Material Design: how to actually build it, per framework

**Material Design is the design language; the library you build it with depends on your app's framework — keep the app in its own framework.** The shared Material *system* (colour, typography, the 8px spacing grid, elevation, shape, theming, and the accessibility contract) is framework-agnostic and lives in `styles.md`, `patterns.md`, and `components.md`. This file covers the **consume layer**: which Material component library to use for each framework, and how to install, theme, and use it.

**Do not flip an app's framework to satisfy the design system.** A SvelteKit app stays SvelteKit and uses a Svelte Material library; a React app uses MUI. The old failure mode this pack fixes is treating "Material" as "React-only" and either rewriting a Svelte app in React or hand-approximating Material with no component library — both are wrong.

## The shared contract (holds for every framework)

Whatever library you use, keep the parts that make it Material and accessible:

1. **Use the library's Material component** rather than hand-rolling a `<button>`/`<input>`/`<div onClick>`. The inventory in `components.md` is the Material component set — each framework's library provides the same components under its own import paths.
2. **Theme once, consume tokens.** Set palette, typography, spacing, shape in one place and reference tokens — never scatter hex or magic pixels (that breaks dark mode and the scale). How tokens are *set and read* differs per library (see `styles.md` for the system, and each framework section below for the API).
3. **You still own accessibility.** The library gives correct roles/focus/keyboard behaviour, but you supply labels, dialog titles, accessible names for icon-only controls, contrast after theming, heading order, and alt text. See the `mui-accessibility` skill.

## Choose the library for your framework

| Framework | Library | Material version | Notes |
| --- | --- | --- | --- |
| **Svelte / SvelteKit** | **SMUI** (Svelte Material UI) — default | Material Design 2 (matches `@mui/material`) | Most-adopted Svelte Material library; Svelte 5 native (v8+); v9 is independent of the deprecated MDC-Web. **Start here.** |
| **Svelte / SvelteKit** | **m3-svelte** — alternative | Material 3 / Material You | Choose when you specifically want Material 3 fidelity; actively maintained but younger and a smaller component set. |
| **React / Next.js** | **@mui/material** (MUI) | Material Design 2 (M3 support experimental) | The original, broadest, batteries-included path. |
| **Vue / Angular / other** | see "Other frameworks" below | varies | Vuetify, Angular Material, or the Material Web components. |

> **Maintenance/fidelity note (verify before relying on this).** This snapshot reflects the state around mid-2026; the Svelte options are fast-moving single-/small-team community projects. Confirm the current major version and Svelte-version support in the target repo before choosing. `@mui/material` and SMUI both render **Material Design 2**; only **m3-svelte** renders **Material 3** — so mixing m3-svelte screens with MUI/SMUI screens in one product will look visually different. Pick one Material generation per product.

---

# Svelte / SvelteKit (the default app template — do this first)

SvelteKit is the default app template, so this is the path most builds need. **Do not port a SvelteKit app to React to get Material** — use a Svelte Material library.

## Which Svelte library

**Default to SMUI.** It is the most widely used Svelte Material library, is actively maintained, supports Svelte 5 (v8+), and — since v9 — no longer depends on Google's deprecated MDC-Web, having absorbed that functionality into its own independent components. It renders **Material Design 2**, which matches `@mui/material` (this pack's React path) and the values documented in `styles.md`, so a Svelte build and a React build from this pack look like the same product.

**Use m3-svelte instead when you specifically want Material 3 (Material You).** It is the actively-maintained Svelte implementation that tracks the current Material 3 spec and design tokens, and supports Svelte 5. Trade-offs: it is younger, has a smaller community and component set than SMUI, and its M3 look diverges visually from the MD2 system the rest of this pack describes. If Material 3 fidelity is a stated requirement, prefer it and adapt the token names accordingly (M3 uses role tokens like `primary`, `on-primary`, `surface`, `surface-container`).

The rest of this section shows **SMUI** (the default). The m3-svelte quickstart is at the end.

## SMUI — install

SMUI ships one package per component plus a small set of core packages. Add SvelteKit's Vite Sass support and the components you use:

```bash
npm install --save-dev sass
npm install @smui/button @smui/textfield @smui/card @smui/dialog @smui/data-table \
            @smui/select @smui/checkbox @smui/radio @smui/list @smui/top-app-bar \
            @smui/drawer @smui/snackbar @smui/tab-bar
```

(Install only the component packages you actually use — each is separate, which keeps the bundle lean. There is no single barrel to import.)

**Roboto + Material Icons** (Material's default typeface/icon set) are not bundled — self-host with `@fontsource/roboto` and `@fontsource/material-icons`, or your own brand font, and import them once in your root layout:

```bash
npm install @fontsource/roboto @fontsource/material-icons
```

```js
// src/routes/+layout.svelte (script)
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';
import '@fontsource/material-icons';
```

## SMUI — theming (set the Material system once)

SMUI themes with **Sass**, not a JS `createTheme`. Create a theme file that sets the Material palette/shape tokens, then compile it into your app's CSS. The idiomatic setup uses SMUI's theme entry points:

```scss
// src/theme/_smui-theme.scss  (light theme)
@use '@material/theme/color-palette';

// Brand tokens — the same roles as the MUI palette (styles.md), set once here.
// Source these from the app's injected `## Brand colours` system block: set
// $primary to the brand PRIMARY hex and $secondary to the brand SECONDARY hex.
// Do NOT ship a default MDC blue — use the app's resolved brand colours.
$primary: <brand primary hex>;      // from the injected "## Brand colours" block
$secondary: <brand secondary hex>;  // from the injected "## Brand colours" block
$error: color-palette.$red-900;
$background: #fff;      // page
$surface: #fff;         // cards/sheets (MUI's background.paper)
$on-primary: #fff;
$on-secondary: #fff;
$on-surface: #000;
```

```scss
// src/theme/dark/_smui-theme.scss  (dark theme — same roles, dark values)
@use '@material/theme/color-palette';
$background: #101418;
$surface: #1c2126;
$on-surface: #fff;
// …
```

Wire SvelteKit's Vite config to resolve `@material`/`@smui` Sass paths and build both themes (SMUI's `smui-theme` CLI, `npx smui-theme template src/theme`, scaffolds this). Reference tokens in your own components with CSS custom properties (`var(--mdc-theme-primary)`) or SMUI's Sass mixins — **never hardcode the hex again**. This is the SMUI equivalent of "consume `theme.palette.*`, don't hardcode": the palette/shape/type values from `styles.md` are set here once and every component reads them.

Dark mode: compile a dark theme (as above) and switch stylesheets (or toggle a `class`/`data-theme` on `<html>` and scope the dark tokens under it); seed the initial choice from `prefers-color-scheme` and persist it. Guard any `localStorage`/`window` access with `typeof window !== 'undefined'` in SvelteKit.

## SMUI — using components

Import each component from its package and use it as a Svelte component. The component names and accessibility requirements mirror the Material inventory in `components.md`:

```svelte
<script>
  import Button, { Label } from '@smui/button';
  import Textfield from '@smui/textfield';
  import HelperText from '@smui/textfield/helper-text';
  import IconButton from '@smui/icon-button';

  let email = $state('');
</script>

<Textfield bind:value={email} label="Email" type="email" input$autocomplete="email">
  {#snippet helper()}
    <HelperText>We'll never share it.</HelperText>
  {/snippet}
</Textfield>

<Button variant="raised" onclick={submit}>
  <Label>Save</Label>
</Button>

<!-- icon-only control MUST carry an accessible name -->
<IconButton class="material-icons" aria-label="Close">close</IconButton>
```

- **Variants map to MUI:** SMUI `variant="raised"` ≈ MUI `contained`; `"outlined"` ≈ `outlined`; text button (no variant) ≈ MUI `text`.
- **`Textfield`** takes `label`, `type`, and a `HelperText` slot — the same contract as MUI's `TextField`/`helperText`. Pass input attributes with the `input$` prefix (`input$autocomplete`, `input$inputmode`).
- **Icon-only `IconButton`/`Fab`** must have an `aria-label` — same rule as MUI (WCAG 4.1.2).
- **Dialogs** (`@smui/dialog`) need a labelled title region (`aria-labelledby`) and keep focus-trap/Esc behaviour — don't strip it.
- Prefer the SMUI component over hand-rolled markup, exactly as `components.md` says for MUI.

## SMUI — SvelteKit specifics

- **SSR / `adapter-static`:** SMUI works with SvelteKit SSR and with the SPA (`adapter-static`, `ssr = false`) setup. Because styling is compiled CSS (not runtime CSS-in-JS), there is no flash-of-unstyled-content problem to special-case the way MUI+Emotion needs on Next.js.
- **Vite Sass:** ensure `sass` is installed and Vite can resolve `@material`/`@smui` includes (the `smui-theme` scaffold sets the include paths). Compile the theme CSS as part of the build.
- **Runes:** use Svelte 5 runes (`$state`, `$derived`, `$effect`) in your own code, as the surrounding SvelteKit app does.

## m3-svelte — quickstart (the Material 3 alternative)

```bash
npm install m3-svelte
```

m3-svelte uses **Material 3 design tokens** (CSS custom properties like `--md-sys-color-primary`, `--md-sys-color-surface`). Generate a theme from a seed/brand colour with its theming utilities, apply the token CSS at the root, then use its components:

```svelte
<script>
  import { Button, TextField } from 'm3-svelte';
</script>

<TextField label="Email" type="email" bind:value={email} />
<Button variant="filled" onclick={submit}>Save</Button>
```

- Tokens are **Material 3 role tokens** (`primary`/`on-primary`/`surface`/`surface-container`/…) — a richer, different set from MD2's `primary.main`/`background.paper`. When following `styles.md`, map MD2 roles to their M3 equivalents.
- Dark mode is a token swap (light/dark schemes generated from the same seed).
- Confirm the current component coverage against the m3-svelte docs — it is smaller than SMUI/MUI, so some patterns in `patterns.md` may need composing from primitives.

---

# React / Next.js — @mui/material (MUI)

MUI (`@mui/material`) is the React implementation of Material Design, styled with **Emotion** by default. It renders Material Design 2 (its Material 3 support is experimental). This is the original, broadest path.

## Installing

```bash
npm install @mui/material @emotion/react @emotion/styled
```

Icons and the MUI X packages are separate:

```bash
npm install @mui/icons-material
npm install @mui/x-data-grid          # data grid
npm install @mui/x-date-pickers dayjs # date/time pickers (+ a date adapter)
npm install @mui/x-charts             # charts
```

**Roboto** is not bundled — self-host via `@fontsource/roboto` (or your own brand font set in `typography.fontFamily`):

```bash
npm install @fontsource/roboto
```

```js
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';
```

(styled-components is a supported alternative engine, but Emotion is the default and simplest — pick one and stay consistent.)

## Theme provider and baseline (set up once)

Wrap the app in a single `ThemeProvider` and render `CssBaseline` once:

```jsx
import { createTheme, ThemeProvider, CssBaseline } from '@mui/material';

const theme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: '#1976d2' },
    secondary: { main: '#9c27b0' },
  },
  typography: { fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif' },
  shape: { borderRadius: 8 },
});

export default function App() {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      {/* app */}
    </ThemeProvider>
  );
}
```

`CssBaseline` normalises browser styles and applies the theme's background/text colours — use it instead of a hand-written reset. For dark mode, drive `palette.mode` from state (persist the choice); or adopt MUI's CSS-variables theming (`cssVariables: true` + `colorSchemes`) to support both schemes without a flash — confirm the API for your version.

## Styling: sx, styled, and the theme

Three idiomatic ways to style, all reading from the theme:

- **`sx` prop** — one-off, co-located styling with theme-aware shorthands and responsive objects:
  ```jsx
  <Box sx={{ p: 2, mt: 3, bgcolor: 'background.paper', color: 'text.secondary',
             display: { xs: 'block', md: 'flex' }, gap: 2 }} />
  ```
- **`styled()`** — reusable styled components:
  ```jsx
  import { styled } from '@mui/material/styles';
  const Hero = styled('section')(({ theme }) => ({
    padding: theme.spacing(4),
    background: theme.palette.primary.main,
    color: theme.palette.primary.contrastText,
    borderRadius: theme.shape.borderRadius,
  }));
  ```
- **Theme `components` overrides** — set default props / style overrides for a component library-wide (see `styles.md`).

Always consume tokens (`theme.palette.*`, `theme.spacing`, `theme.typography.*`, `theme.shape`) — never hardcode hex/px. In `sx`, numeric `p`/`m`/`gap` are spacing multiples; colour strings like `'primary.main'` resolve against the palette.

## Imports and bundle size

Prefer **per-path imports** so bundlers tree-shake cleanly:

```js
import Button from '@mui/material/Button';
import TextField from '@mui/material/TextField';
```

Top-level named imports (`import { Button } from '@mui/material'`) also work and modern bundlers tree-shake them, but per-path is the safe default. Icons are always per-path: `import DeleteIcon from '@mui/icons-material/Delete'` (the barrel is huge). Do not import the whole icon package.

## Icons

Material icons ship as React components in `@mui/icons-material` in five themes (Filled, Outlined, Rounded, TwoTone, Sharp — suffix the name, e.g. `DeleteOutlined`). A **decorative** icon should be hidden from assistive tech; an icon that carries meaning needs a text alternative — usually via the surrounding control's `aria-label` (an `IconButton` with only an icon must have one).

## Using MUI with React frameworks

- **Vite / CRA (SPA):** the `ThemeProvider` + `CssBaseline` setup above is all you need. Emotion works out of the box.
- **Next.js (App Router):** MUI components are client components — mark files `'use client'` where needed and use MUI's Next.js integration (`@mui/material-nextjs`) so Emotion's styles are injected correctly during SSR (avoids a flash of unstyled content). Put `ThemeProvider`/`CssBaseline` in a client root and, for the App Router, use `AppRouterCacheProvider`.
- **TypeScript:** MUI is fully typed. To add custom theme tokens (extra palette entries, custom variants), augment the theme types via module augmentation so `theme.palette.myBrand` is typed.

---

# Other frameworks

Material Design is not React-only, and not Svelte-only. For other stacks, use that framework's Material component library and keep the same shared contract (Material component, theme once, own the a11y):

- **Vue:** **Vuetify** (<https://vuetifyjs.com/>) — a mature, broad Material component framework for Vue, themeable to brand.
- **Angular:** **Angular Material** (<https://material.angular.io/>) — Google's own Angular implementation, tracking Material 3.
- **Framework-agnostic / web components:** **Material Web** (`@material/web`, <https://github.com/material-components/material-web>) — Google's Material 3 web components, usable from any framework or plain HTML. Useful as a last-resort or for micro-frontends; confirm its current maintenance status, as Google has repeatedly re-scoped its web-component efforts.

For any of these, the values and patterns in `styles.md`/`patterns.md`/`components.md` still apply — only the import paths and the theming API change. If you end up hand-authoring plain HTML/CSS with no Material library, you lose the built-in accessible behaviour (focus management, keyboard interaction, ARIA) and must supply it yourself — which is exactly what these libraries exist to prevent. Prefer a supported library.

# Keeping it idiomatic (any library)

- One theme, one baseline, one styling mechanism — don't mix engines/theme sources.
- Reach for an existing Material component before hand-rolling markup (see `components.md`); reach for a data-grid/date-picker/chart component before hand-building one.
- Theme to brand; don't fork components or fight them with `!important`.
- Pin and track your library's major version — MUI's Grid/styling/theme APIs shifted across v4→v7, SMUI's across MDC-Web → v9-independent, and m3-svelte moves quickly; confirm snippets against the version in `package.json`.

# Coding in the open

MUI, SMUI, m3-svelte, Material Web, and Material Design references are open source (the libraries MIT-licensed; Material Design © Google). Attribute per the NOTICE where you redistribute the bundled reference content; keep secrets and personal data out of the repo.
