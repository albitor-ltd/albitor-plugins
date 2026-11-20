---
name: look-business
description: "Dense, exact and quiet: for enterprise and line-of-business web apps (admin, finance, operations) where staff live in tables all day. Navy neutrals, 2px corners, no elevation, compact rows and tabular figures."
capability: visual-language
---

# Business look

Dense, exact and quiet. Navy neutrals, a single blue accent, 2px corners, compact rows, tabular figures, no shadows.

A look is a pure style layer over the app's chosen component kit and markup. It changes tokens
(type, colour, radius, density, elevation), never the accessibility behaviour of a component. The platform paints the preview from `preview-tokens.json` at this plugin's root and
derives the full light and dark scheme from it; the build must use those derived tokens, not
re-invent them.

## Suits

Apps where exact figures and long sessions matter: admin consoles, finance, back office, operations,
registers, logs, rotas, bookings, inventories and case-work tools, wherever people compare exact
values across many rows. Works with any shell and first screen the app picks. Avoid for mobile-first
consumer or marketing surfaces.

## Apply it

The composed app carries the look's token set as `web/src/lib/look/look.json` (seeded once, a copy
of this plugin's `preview-tokens.json`) and the platform writes `web/src/lib/look/look.css` from
it on every build: the look's colours, radius, spacing, shadow, type and motion as `--look-*`
custom properties, in light and dark. The kit reads them and maps them onto its own tokens, so the
app already looks like this look. **Never edit `look.css`** (the platform overwrites it) and do not
map tokens by hand or redeclare the kit's colour tokens in light and dark selectors. Styles the
app needs beyond the look go in `web/src/lib/app/app.css`, over the kit's tokens; never hardcode a
hex in a component. The kit offers dark and the runner scans both themes.

This look suggests the **Technical** font look (`default_font_look` in the manifest) when the person
has not picked one; their pick and any policy lock outrank it.
The look also supplies page examples (below) and a `rubric.md` for the visual check.

### Typography

- System UI stack led by Segoe UI (`font_family` in `preview-tokens.json`); one family throughout, headings differ by size and weight 600 only.
- Small type is deliberate: body 0.8125rem, secondary 0.75rem, section heading 0.9375rem, page title 1.125rem. Do not scale up to fill space; add columns instead.
- Turn on tabular figures (`font-variant-numeric: tabular-nums`) everywhere numbers are compared. Right-align numeric columns; align their headers the same way.

### Density

- Compact. Table rows and form controls sit at a 28px minimum target height; padding steps are 4/8/12px, not 16/24.
- Prefer one dense table with filters above it to a stack of cards. Keep line length and column order stable between views so users build muscle memory.

### Shape and elevation

- 2px radius on controls, panels and pills alike; 1px borders; never fully round a control.
- Elevation is none. Separate regions with a 1px `border` line or a step to `surface`; menus and dialogs may carry a 1px border and nothing softer. Totals are marked with a stronger border, not a brighter colour.

### Colour

- Neutrals are navy-tinted: text `#0b1b33`, page `#f8fafc`, panel `#e9eef5`. The accent `#0b4f9e` is used for primary actions, links and the selected state only, never as a large fill.
- Status colours (danger, success, warning) carry meaning only in combination with a glyph and a word, never hue alone. Selected rows use a tint plus a 2px left bar.
- Controls use `control_border` (3:1 against the page); decorative dividers use the quieter `border`.

The authored light palette: text `#0b1b33`, secondary text `#44546a`, border
`#aebbcc`, control border `#7a8aa0`, surface `#e9eef5`, page
`#f8fafc`, accent `#0b4f9e` with `#ffffff` on it. The accent is a default:
the app's own brand colour replaces it and the platform recalculates the dependent shades.

## Signature

What makes an app in this look recognisable. These are positive recipes; build them in, and review a built page against `rubric.md`.

- **Second colour:** carries selection and emphasis in the data, never an action: the tint on a selected row and the bulk bar, the current item in the nav, `Card tone="second"`. The blue accent stays on primary actions, links and the selected state.
- **Icons:** Lucide through the kit's `Icon`, small (14 to 16px), one beside every nav destination; row and cell content stays text.
- **Navigation styling:** the look styles whatever shell the structure picks; it never picks one.
  Side: a compact column, 28px rows, the selected item a tint plus a 2px left bar. Top: a compact
  bar separated by a 1px line, the current link a tint plus a 2px bottom bar. Tabs: a 1px top line,
  compact labels, the current tab a tint plus a 2px top bar. Hub: plain bordered tiles of exact
  values, no decoration. None: no chrome, the table or form starts at the top. No shadow in any of
  them.
- **Figures:** headline figures wherever they appear are one bordered strip of exact values with a
  note under each, in tabular figures. Not large decorated tiles.
- **Lists:** one dense table with saved views and a search field above it, numbers right-aligned, and a bulk bar that appears when rows are selected (a count and the actions that apply). A selected row is a tint plus a 2px left bar.
- **Art:** the ruled-page `EmptyArt`, small, on a first-run page only; in a dense table the plain sentence is the empty state.
- **Motion:** none beyond state changes. A short fade for a dialog or a toast and a visible focus ring; nothing moves for decoration.

The signature is style, not structure: the kit's page templates and the app's own layout are unchanged, and a look never changes accessibility behaviour. Structure (`layout`, `phoneNav`) and style (the tokens) are separate axes: another look can use the same shell props.

## Empty-state art

The look's art is seeded as `web/src/lib/look/EmptyArt.svelte`: a starting point, then the app's
own, so edit it freely. It is drawn in the app's tokens, so it follows the primary, second and
surface colours and dark mode with no second asset. Use it for first-run states; the end-of-build
prune removes it if nothing imports it.

- **Use it** on a first-run empty list or page (nothing created yet), above the plain sentence that
  says why it is empty and the one action that fixes it. The sentence and the action do the work;
  the art is decoration.
- **Do not use it** for errors or failed loads (say what went wrong and how to retry, with a status
  glyph and a word), for an empty search or filter result, inside dense admin tables or a data-heavy
  view, or in a GDS or Carbon app, whose kit has its own conventions and no illustration.
- **Build it** as an inline SVG Svelte component of simple geometric shapes: no lettering, no
  photographic or detailed drawing, `aria-hidden="true"` and `focusable="false"`, no `<title>`.
  Fill and stroke only with the app's colour tokens (`var(--primary)`, `var(--second)`,
  `var(--muted)`, `var(--border)` on the shadcn kit; the app's own names otherwise), never a hex, so it follows the
  light and dark pages with no second asset. Use opacity for tints, not new colours. It never
  carries the only cue for anything.
- **Look-specific:** shapes are rectilinear and quiet: a ruled page, a table or a form outline with
  2px corners, straight 2px lines, one accent bar. Keep it small (about 144px) on a calm surface. In
  a dense table or multi-column admin view the plain sentence alone is the empty state.

## Phone navigation

Stance (the fallback when the structure gives no phone navigation): lean towards a menu button and
drawer, which keeps the dense, table-first screen uncluttered.

Rule (shared with the kit): 2 to 5 frequent top-level destinations use a bottom tab bar; more than
5 use a menu button and drawer, or, where this look prefers a tab bar, 4 tabs plus a "More" tab that
opens the rest as a list. Pick one per app, never a tab bar and a menu button carrying the same
destinations. This is a preference, not a mandate. Follow the kit's guidance on markup and behaviour.

- Menu button: a `Dialog.Trigger` (real `<button>`, accessible name, `aria-expanded` and
  `aria-controls` wired by Bits) opening a modal drawer with a `Dialog.Title`, containing a `<nav>`.
  Focus moves in on open; Escape closes and returns focus to the button; focus is trapped and the
  background inert.
- Tab bar: a labelled `<nav>`, `aria-current="page"` on the current tab, targets at least 44x44 CSS px
  (WCAG 2.2 2.5.8 requires 24x24; 44 is the house recommendation) and
  `env(safe-area-inset-bottom)` padding.
- More: never `aria-current` on the More button; keep it on the item in the opened list, and mark
  More visually plus with hidden text such as "(current page is in More)".
- Never hide a control: every desktop nav destination stays reachable on phone. Test at 390px.
- Tab and nav targets stay at least 44px, overriding the look's 28px density minimum. Keep the bar
  compact, with a 1px `border` top line and no shadow.

### Wire it

The shell is props, not prose to remember. In the root `web/src/routes/+layout.svelte`:

1. Pass `layout` and `phoneNav` to `AppShell` from the structure's shell directive (the app's pick
   owns the shell; see the kit-shadcn SKILL "Navigation: structure and style are separate" for how
   each directive maps to these props, and for `hub` and no shell). Where the build was given no
   directive, fall back to this look's stance above: `phoneNav="drawer"` and `layout="side"`. Never
   choose the shell from the look.
2. Build an `icons` map from the app's own nav, `href` to Lucide component, and pass it too. Import
   each icon on its own from `@lucide/svelte/icons/<name>` (never the barrel), give every
   destination one, and update the map when a route is added, renamed or removed (a missing entry
   shows the label alone).

`examples/web/src/routes/+layout.svelte` beside this skill shows the whole layout wired this way.
Keep the app's own providers, title and mark; add only `layout`, `phoneNav` and `icons`.

## Page examples

Styled examples of every kit page template are in `examples/web/src/lib/look/pages/`, laid out by
repository path: Overview (summary strip, a table and an ageing card), Invoices (list with saved views, search, row selection and a bulk bar), Invoice (detail with a status, lines and a confirm dialog), NewInvoice (form), Preferences (settings), FirstRun and Setup (a step flow with a numbered list of steps). `pages.test.ts` renders each page (loading, failed and
empty states included) and runs axe, one `h1` and heading-order checks over it.

They are worked examples, not seeded files. Copy the ones the app needs, with `pages.test.ts`
trimmed to match, to `web/src/lib/look/pages/`, then edit them: replace the sample data and copy,
fetch in `+page.ts` or the page, and pass `status`. They import the kit's page templates
(`DashboardPage`, `ListPage`, `DetailPage`, `FormPage`, `SettingsPage`, `PageHeader`,
`StateNotice`), which come from the `kit-shadcn` skill's examples, so copy those first. Once
copied they are the app's code and never reference this skill. Do not rebuild a page from scratch
when one of these fits.

## Reviewing a page

`rubric.md` beside this file is a short checklist for a visual review of a built page against this
look (the heading font loaded, the page colour from `look.css`, the second colour in the data,
icons in the nav, a designed first-run state, density). Use it on screenshots or a rendered page.

## Dark mode

- Page near #0b1219, surfaces one step lighter, 1px borders carry the separation (shadows vanish in dark).
- Body text capped near #e1e8f2; accent used for links and primary only, no large fills.
- Status colours lightened, always with a label and glyph. Dense tables: faint dividers, no zebra, selected row = tint plus a 2px left bar.
- Numbers right-aligned and tabular; totals marked by a stronger border, not a brighter colour.
- Dark accent. The light accent fails 4.5:1 on the dark page, so on dark pages (`#09090b` and the
  panel steps above it) use `#5aa9f0` for links, accent text and the primary fill (dark ink `#09090b`
  on it); it holds 4.5:1 or better on `#09090b`, `#18181b` and `#27272a`.

## Accessibility floor

The look never lowers the WCAG 2.2 AA floor: body text 4.5:1, control borders and the focus ring
3:1, a visible focus ring on every interactive element, and status never conveyed by colour alone.
If a brand colour cannot meet the floor as written, shift the shade, do not break the contrast.
