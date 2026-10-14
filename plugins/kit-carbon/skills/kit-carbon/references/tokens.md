# Tokens

Carbon's themes are CSS custom properties, `--cds-*`, loaded by `AppShell` from
`carbon-components-svelte/css/all.css`: `:root` is the light theme (`white`) and `[theme=g100]` the
dark one. The kit adds none and is not themeable, so read them; do not override them.

| Token | Role |
|---|---|
| `--cds-background` | Page background |
| `--cds-layer` | A panel on the page (empty state, table header) |
| `--cds-text-primary`, `--cds-text-secondary` | Body text; secondary text |
| `--cds-text-helper` | Hint text under a field |
| `--cds-text-error`, `--cds-support-error` | Error text; error edges and icons |
| `--cds-text-on-color` | Text on a button fill |
| `--cds-border-strong` | A control's edge. 3:1 on the background |
| `--cds-border-subtle` | Decorative hairlines (table rules). Not for a control edge |
| `--cds-focus` | Focus outline. 3:1 on the background |
| `--cds-link-primary` | Link text |
| `--cds-button-primary`, `--cds-button-secondary`, `--cds-button-danger-primary` | Button fills |
| `--cds-layer-selected` | The selected row or item |

Pairs audited by `contrast.test.ts`, read from the installed stylesheet, in the light and dark
theme:

- text 4.5:1: text-primary on background, layer and layer-selected; text-secondary and text-helper
  on background and layer; text-error and link-primary on background; text-on-color on the three
  button fills.
- non-text 3:1: border-strong, focus and button-primary against the background.

If you add a token pair to a component, add it to the `pairs` list in the test in the same change.
`border-subtle` is decorative and exempt, so keep any border that identifies a control on
`--cds-border-strong`. `preview-tokens.json` (the plugin root, read by the platform for the build
preview) carries Carbon white's palette.

`theme.ts` chooses between `white` and `g100`: the visitor's `prefers-color-scheme`, or
`<html data-theme="light" | "dark">` when the page sets it. The other Carbon themes (`g10`, `g80`,
`g90`) are not used and are not audited.
