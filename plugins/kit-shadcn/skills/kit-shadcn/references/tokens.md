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
| `--muted`, `--muted-foreground` | `surface`, `ink-soft` | Panels, table headers, hover; secondary text and hints |
| `--border` | `line` | Decorative hairlines (table rules, header rule). Not for a control edge |
| `--input` | `control-line` | A control's edge (field, secondary button, dashed empty state). 3:1 on the background |
| `--ring`, `--focus-width` | `focus`, `focus-width` | Focus outline colour (3:1 on the background and the primary fill) and thickness |
| `--primary`, `--primary-foreground` | `accent`, `on-accent` | The main action fill and its text |
| `--second`, `--second-subtle`, `--second-text`, `--on-second` | `second*`, `on-second` | The second colour: fill, chip background, text on page and chip, ink on the fill. With no second colour they equal the primary and muted |
| `--destructive`, `--destructive-foreground` | `danger`, `on-danger` | Delete, errors, and the text on a destructive fill |
| `--success`, `--on-success`, `--warning`, `--on-warning` | `success`, `warning`, `on-*` | Status fills (success toast edge) and their ink |
| `--link` | `link`, else `accent-text` | Link text |
| `--header`, `--header-ink` | `header`, `header-ink` | The app header band (the page colour with no look) |
| `--radius`, `--border-width` | `radius`, `border-width` | Corner radius; border thickness (default `1px`) |
| `--shadow`, `--shadow-overlay` | `shadow` | Raised surfaces (header, buttons, stacked rows); the dialog and toasts. `none` with no look |
| `--font-sans`, `--text-small`, `--text-body`, `--text-heading`, `--text-title` | `font-body`, `text-*` | Type |
| `--heading-font`, `--heading-weight`, `--heading-tracking` | `font-heading`, `weight-bold`, `tracking` | Headings, table captions, labels. The names avoid `--font-heading`, which the font plugins own |
| `--figures` | `figures` | `font-variant-numeric` for body and tables |
| `--space`, `--space-1`, `-2`, `-3`, `-4`, `-6`, `-8` | `space` | Density multiplier and the spacing steps (0.25, 0.5, 0.75, 1, 1.5, 2 rem times it) |
| `--control-height`, `--row-height` | `min-target`, `space` | Button and field minimum height; table row height |
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
