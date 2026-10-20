---
name: look-business
description: "Dense, exact and quiet: for enterprise and line-of-business web apps (admin, finance, operations) where staff live in tables all day. Navy neutrals, 2px corners, no elevation, compact rows and tabular figures."
capability: visual-language
---

# Business look

Dense, exact and quiet. Navy neutrals, a single blue accent, 2px corners, compact rows, tabular figures, no shadows.

A look is a pure style layer over the app's chosen component kit and markup. It changes tokens
(type, colour, radius, density, elevation), never the structure or the accessibility behaviour of a
component. The platform paints the preview from `preview-tokens.json` at this plugin's root and
derives the full light and dark scheme from it; the build must use those derived tokens, not
re-invent them.

## Suits

Enterprise and line-of-business apps: admin consoles, finance, back office and operations tools used by staff for long sessions, where comparing exact numbers across many rows matters more than first impressions. Avoid for mobile-first consumer or marketing surfaces.

## Apply it

The composed app carries the look's token set as `web/src/lib/look/look.json` (seeded once, a copy
of this plugin's `preview-tokens.json`). Map its values onto the kit's colour, radius and type tokens
in one place (declare them unlayered in `web/src/lib/app/app.css`, which wins over the kit's
`albitor-kit` layer); never hardcode a hex in a component. Dark values are derived, not authored: keep
the dark scheme the kit already ships unless the platform hands you derived values.

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

## Dark mode

- Page near #0b1219, surfaces one step lighter, 1px borders carry the separation (shadows vanish in dark).
- Body text capped near #e1e8f2; accent used for links and primary only, no large fills.
- Status colours lightened, always with a label and glyph. Dense tables: faint dividers, no zebra, selected row = tint plus a 2px left bar.
- Numbers right-aligned and tabular; totals marked by a stronger border, not a brighter colour.

## Accessibility floor

The look never lowers the WCAG 2.2 AA floor: body text 4.5:1, control borders and the focus ring
3:1, a visible focus ring on every interactive element, and status never conveyed by colour alone.
If a brand colour cannot meet the floor as written, shift the shade, do not break the contrast.
