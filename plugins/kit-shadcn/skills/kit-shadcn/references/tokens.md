# Tokens

All in `web/src/lib/kit/tokens.css` (the defaults), in the `albitor-kit` cascade layer. Override by
declaring the same property unlayered in `web/src/lib/app/app.css`.

| Token | Role |
|---|---|
| `--background`, `--foreground` | Page and body text |
| `--muted`, `--muted-foreground` | Panels, table headers; secondary text and hints |
| `--border` | Decorative hairlines (table rules, header rule). Not for a control edge |
| `--input` | A control's edge (field, secondary button, dashed empty state). 3:1 on the background |
| `--ring` | Focus outline. 3:1 on the background and on the primary fill |
| `--primary`, `--primary-foreground` | The main action fill and its text |
| `--destructive`, `--destructive-foreground` | Delete, errors, and the text on a destructive fill |
| `--link` | Link text |
| `--radius` | Corner radius |
| `--font-sans`, `--text-small`, `--text-body`, `--text-heading`, `--text-title` | Type |

Pairs audited by `contrast.test.ts`, in light, dark by preference and dark by attribute:

- text 4.5:1: foreground on background and on muted; muted-foreground on background and on muted;
  primary-foreground on primary; destructive-foreground on destructive; destructive and link on
  background.
- non-text 3:1: input, ring, and primary against the background; ring against primary.

If you add a token pair to a component, add it to the `pairs` list in the test in the same change.
`border` is decorative and exempt, so keep any border that identifies a control on `--input`.
`preview-tokens.json` (the plugin root, read by the platform for the build preview) carries the same
light palette; keep them equal.
