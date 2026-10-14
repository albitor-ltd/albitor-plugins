Source: Material UI (MUI) documentation (https://mui.com/) and Material Design (https://m3.material.io/, https://m2.material.io/). MUI is MIT-licensed; Material Design is © Google. Distilled reference — verify against the live source for current detail.

# Material styles: the shared Material system and theming

**This is the framework-agnostic Material system** — the values (colour roles, type scale, 8px spacing grid, elevation, shape) are the same whether you build with SMUI, m3-svelte, or `@mui/material`. Only the *API for setting and reading them* differs per library (see `frontend-conventions.md`): MUI React uses `createTheme` + tokens like `theme.palette.*`; SMUI uses a Sass theme compiled to CSS custom properties (`var(--mdc-theme-*)`); m3-svelte uses Material 3 design tokens (`--md-sys-color-*`). The code below is shown in the **`@mui`/React** form as the reference idiom.

> **Material generation:** the default values here are **Material Design 2** (as rendered by `@mui/material` and SMUI). **m3-svelte renders Material 3**, whose token *names* and some values differ (richer role tokens: `primary`/`on-primary`/`surface`/`surface-container`). The *principles* — set once, consume tokens, meet contrast — hold across both.

The golden rule (every library): **set these once in a theme and consume tokens** — never hardcode hex or magic pixels, or you break dark mode and the scale.

## The theme object (MUI React form)

`createTheme({...})` returns the theme; `<ThemeProvider theme={theme}>` makes it available; `<CssBaseline />` applies its baseline. Access it via the `sx` prop (`sx={{ color: 'primary.main', p: 2 }}`), `styled((theme) => …)`, or the `useTheme()` hook. Key branches: `palette`, `typography`, `spacing`, `shape`, `shadows`, `breakpoints`, `zIndex`, `transitions`, and `components` (default props + style overrides per component). In SMUI the equivalent is the Sass theme file; in m3-svelte, the generated M3 token set — same roles, different plumbing (see `frontend-conventions.md`).

## Colour

### How to use colour in code
Reference palette tokens, not hex: `theme.palette.primary.main`, `.light`, `.dark`, `.contrastText`; the same for `secondary`, `error`, `warning`, `info`, `success`; plus `text.primary`/`text.secondary`/`text.disabled`, `background.default`/`background.paper`, `divider`, and `action.*` (hover/selected/disabled). In `sx` you can use the short string form: `sx={{ bgcolor: 'background.paper', color: 'text.secondary' }}`.

### The default palette (indicative — set your own brand values in the theme)
| Role | Token | Light-mode default (approx.) |
| --- | --- | --- |
| Primary | `palette.primary.main` | `#1976d2` (blue) |
| Secondary | `palette.secondary.main` | `#9c27b0` (purple) |
| Error | `palette.error.main` | `#d32f2f` |
| Warning | `palette.warning.main` | `#ed6c02` |
| Info | `palette.info.main` | `#0288d1` |
| Success | `palette.success.main` | `#2e7d32` |
| Text primary | `palette.text.primary` | `rgba(0,0,0,0.87)` |
| Text secondary | `palette.text.secondary` | `rgba(0,0,0,0.6)` |
| Background (page) | `palette.background.default` | `#fff` (light) |
| Background (surface) | `palette.background.paper` | `#fff` |
| Divider | `palette.divider` | `rgba(0,0,0,0.12)` |

Set your brand by overriding these with the app's resolved brand colours from the injected `## Brand colours` system block — set `primary.main` to the brand PRIMARY hex and `secondary.main` to the brand SECONDARY hex: `createTheme({ palette: { primary: { main: '<brand primary hex>' }, secondary: { main: '<brand secondary hex>' } } })`. Never ship the default MDC blue. MUI derives `light`/`dark`/`contrastText` automatically, or you can set them explicitly.

### Light and dark mode
Switch with `palette.mode: 'light' | 'dark'`. In dark mode `background.default`/`paper` go dark and text inverts; because everything reads from tokens, components adapt with no per-component change. Offer a toggle and persist it; seed the initial value from `prefers-color-scheme`. (MUI's newer CSS-variables theming, `cssVariables: true` / `colorSchemes`, lets both schemes coexist without a flash — confirm support for the version in use.)

### Contrast (WCAG)
Meet WCAG 2.2 AA 1.4.3: text contrast ≥ 4.5:1 (≥ 3:1 for large text ≥ 18.66px bold / 24px regular) and ≥ 3:1 for UI components and meaningful graphics (1.4.11). MUI derives `contrastText` for palette colours, but **verify** contrast after you rebrand — a brand colour with white text can fall below 4.5:1. Never use colour as the only signal (1.4.1): pair `color="error"` with text or an icon.

## Typography

MUI's type scale (Material) maps to `theme.typography` variants, rendered by `<Typography variant=…>`:

| Variant | Typical use | Default size (approx.) |
| --- | --- | --- |
| `h1` | Largest heading (hero) | ~96px / 6rem light |
| `h2` | | ~60px / 3.75rem |
| `h3` | | ~48px / 3rem |
| `h4` | Page heading | ~34px / 2.125rem |
| `h5` | Section heading | ~24px / 1.5rem |
| `h6` | Sub-heading / card title | ~20px / 1.25rem |
| `subtitle1` / `subtitle2` | Supporting titles | ~16px / ~14px |
| `body1` | Default body text | 16px / 1rem |
| `body2` | Secondary body text | 14px / 0.875rem |
| `button` | Button labels | 14px, uppercase by default |
| `caption` | Small print / captions | 12px |
| `overline` | Labels above content | 12px, uppercase, tracked |

- Set the font once: `typography: { fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif' }`. Material's default is **Roboto** (load it yourself; it isn't bundled).
- **Separate visual size from semantics:** `<Typography variant="h4" component="h1">` looks like h4 but is an `<h1>`. Keep one logical `<h1>` per page and a correct heading order (1.3.1/2.4.6) regardless of visual size.
- Use `rem`-based sizes (the defaults are) so text scales with the user's font-size setting (1.4.4).

## Spacing

MUI uses an **8px base spacing unit**. `theme.spacing(n)` returns `n * 8px` (`spacing(2)` = 16px). Shorthand in `sx`: `p`, `px`, `py`, `pt/pr/pb/pl`, `m`, `mx`, `gap`, etc. take spacing multiples — `sx={{ p: 2, mt: 3 }}` = 16px padding, 24px top margin. `Stack spacing={2}` and `Grid spacing={2}` use the same unit. Override the base with `createTheme({ spacing: 4 })` if you need a tighter grid. Prefer these over raw px so rhythm stays consistent.

## Elevation and shadows

Material conveys depth with elevation. `theme.shadows` is an array of 25 shadow definitions (index `0`–`24`); `<Paper elevation={n}>`, `<Card>`, `AppBar`, `Dialog`, `Menu` etc. use them. Higher elevation = more prominent/closer. Keep elevation meaningful and consistent (e.g. app bar and dialogs higher than resting cards); don't let elevation be the *only* way grouping is conveyed — structure must still be semantic. `variant="outlined"` gives a flat, border-only alternative to a shadow.

## Shape (border radius)

`theme.shape.borderRadius` (default 4px) sets the corner radius used across components. Override once (`shape: { borderRadius: 8 }`) to make the whole UI rounder/sharper. Reference it (`sx={{ borderRadius: 1 }}` = 1× the shape radius, or `theme.shape.borderRadius`) instead of hardcoding px.

## Breakpoints and responsive layout

Default breakpoints: `xs` 0, `sm` 600px, `md` 900px, `lg` 1200px, `xl` 1536px. Use them for responsive values:
- Responsive `sx`: `sx={{ width: { xs: '100%', md: 360 }, p: { xs: 2, md: 4 } }}`.
- Responsive `Grid`: size props per breakpoint.
- In JS: `useMediaQuery(theme.breakpoints.up('md'))`.
Design mobile-first and let layouts reflow to a single column so content works at 400% zoom / 320px width without horizontal scrolling (1.4.10).

## Customising components centrally

Set per-component defaults and style overrides in the theme's `components` key instead of repeating props:
```js
createTheme({
  components: {
    MuiButton: {
      defaultProps: { disableElevation: true },
      styleOverrides: { root: { textTransform: 'none' } },
    },
  },
});
```
This keeps every `Button` consistent (e.g. sentence-case labels) without touching each call site. Prefer this and theme tokens over one-off `sx` overrides or `!important`.

## Focus and interaction states

MUI components ship visible focus indicators, ripple/hover states, and correct disabled styling — **keep them**. Never strip focus outlines (`outline: none`) without an equally visible replacement; focus visibility is required by 2.4.7. Rely on the components' built-in states rather than rolling your own, and make sure interactive targets meet the 24×24px minimum (2.5.8) — the default control sizes do.
