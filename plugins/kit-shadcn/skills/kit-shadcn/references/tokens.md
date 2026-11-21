# Tokens

All in `web/src/lib/kit/tokens.css` (the defaults), in the `albitor-kit` cascade layer. The platform
writes `web/src/lib/look/look.css` (`--look-*` properties, never edited by you) and `tokens.css`
imports it: every token reads its `--look-*` property, with the value below as the fallback. To
change something the look does not set, declare the same property unlayered in
`web/src/lib/app/app.css`.

| Token | `--look-*` source | Role |
|---|---|---|
| `--background`, `--foreground` | `page`, `ink` | Page and body text |
| `--surface` | `surface` | Cards, dialog, toasts and field backgrounds (the page colour with no look) |
| `--muted`, `--muted-foreground` | `surface` (so `--muted` and `--surface` share one look value), `ink-soft` | Panels, table headers, hover; secondary text and hints |
| `--border` | `line` | Decorative hairlines (table rules, header rule). Not for a control edge |
| `--input` | `control-line` | A control's edge (field, secondary button). 3:1 on the background |
| `--ring`, `--focus-width` | `focus`, `focus-width` | Focus outline colour (3:1 on the background and the primary fill) and thickness |
| `--primary`, `--primary-foreground` | `accent`, `on-accent` | The main action fill and its text |
| `--second`, `--second-subtle`, `--second-text`, `--on-second` | `second*`, `on-second` | The second colour: fill, chip background, text on page and chip, ink on the fill. With no second colour they equal the primary and muted |
| `--success-edge` | `success` | The success toast's edge: the look's success colour, else `--link` |
| `--destructive`, `--destructive-foreground` | `danger`, `on-danger` | Delete, errors, and the text on a destructive fill |
| `--success`, `--on-success`, `--warning`, `--on-warning` | `success`, `warning`, `on-*` | Status fills (success toast edge) and their ink |
| `--link` | `link`, else `accent-text` | Link text |
| `--header`, `--header-ink` | `header`, `header-ink` | The app header band (the page colour with no look) |
| `--radius`, `--border-width` | `radius`, `border-width` | Corner radius; border thickness (default `1px`) |
| `--shadow`, `--shadow-overlay` | `shadow` | Raised surfaces (header, buttons, stacked rows); the dialog and toasts. `none` with no look, and `none` on the dark page for every look. A `ghost` Button never takes one |
| `--font-sans` | `font-body` | The body face |
| `--text-micro`, `--text-small`, `--text-body`, `--text-lead`, `--text-heading`, `--text-title` | `text-*` | The six-step type scale. The look sets small, body, heading and title; micro (badges, ring values) and lead (a card or sub-section heading) are derived from them. Heading levels step by at least 1.15x (`scripts/check-type-scale.sh`). Fallbacks: 0.75, 0.875, 1, 1.1875, 1.375, 1.875 rem |
| `--leading-micro`, `-small`, `-body`, `-lead`, `-heading`, `-title` | `leading-*` | Unitless line-height for each step: 1.4, 1.4, 1.5, 1.4, 1.3, 1.2. `body` carries `line-height: var(--leading-body)` |
| `--h1-size` to `--h6-size`, `--h1-leading` to `--h6-leading` | `h1-size` and so on | Heading levels onto the steps: h1 title, h2 heading, h3 lead, h4 body, h5 and h6 small. Plain `h1` to `h6` read them. A figure or stat value reads `--h2-size`, so it never rivals the page's h1 |
| `--kit-font-data`, `--kit-font-label` | `font-data`, `font-label` | The face for tables and figures, and for labels and captions; each falls back to the body face. `Table` and `StatTile` read them. The names differ from `--font-data` and `--font-label`, which a font plugin binds and `look.css` reads: reusing them would make a cycle and an empty value |
| `--heading-font`, `--heading-weight`, `--heading-tracking` | `font-heading`, `weight-bold`, `tracking` | Headings, table captions, labels (plain `h1` to `h6` read the `--look-*` family, weight and tracking directly, so they stay as the browser draws them without a look). The names avoid `--font-heading`, which the font plugins own |
| `--figures` | `figures` | `font-variant-numeric` for body and tables |
| `--space`, `--space-1`, `-2`, `-3`, `-4`, `-6`, `-8` | `space` | Density multiplier and the spacing steps (0.25, 0.5, 0.75, 1, 1.5, 2 rem times it) |
| `--control-height`, `--row-height` | `min-target`, `space` | Button and field minimum height; table row height (natural height until the look sets a density) |
| `--hit-inset` | derived from `min-target` | Where a look draws a control under 44px (28px, 36px), the `::after` of `Button`, `FilterChip` and tab triggers extends the hit area to 44px by this much above and below. Zero at 44px and over |
| `--motion-duration`, `--motion-ease` | `motion-duration`, `motion-ease` | Transitions (`0s` with no look, and under reduced motion) |

Pairs audited by `contrast.test.ts`, in light, dark by preference and dark by attribute:

- text 4.5:1: foreground on background and on muted; muted-foreground on background and on muted;
  primary-foreground on primary; destructive-foreground on destructive; destructive and link on
  background.
- non-text 3:1: input, ring, and primary against the background; ring against primary.
- the surface, header, second-colour, success and warning pairs the components use (see the test).

If you add a token pair to a component, add it to the `pairs` list in the test in the same change.
`border` is decorative and exempt, so keep any border that identifies a control on `--input`.
`preview-tokens.json` (the plugin root, read by the platform for the build preview) carries the same
light palette; keep them equal.
