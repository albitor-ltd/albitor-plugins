---
name: look-startup
description: "Crisp, quiet and fast: for SaaS and developer-tool products. One confident teal accent on near-neutral surfaces, generous whitespace, a strong type hierarchy with tight display tracking, hairlines and a soft layered shadow."
capability: visual-language
---

# Startup look

Crisp and quiet. Near-achromatic neutrals, one teal accent, 10px corners, hairlines plus a soft layered shadow, hierarchy from type not decoration.

A look is a pure style layer over the app's chosen component kit and markup. It changes tokens
(type, colour, radius, density, elevation), never the structure or the accessibility behaviour of a
component. The platform paints the preview from `preview-tokens.json` at this plugin's root and
derives the full light and dark scheme from it; the build must use those derived tokens, not
re-invent them.

## Suits

SaaS products and developer tools: dashboards, workspaces, settings and onboarding for people who use the product daily and judge it on speed and clarity. Avoid for dense back-office tables (use Business) or mobile-first lifestyle apps (use Consumer).

## Apply it

The composed app carries the look's token set as `web/src/lib/look/look.json` (seeded once, a copy
of this plugin's `preview-tokens.json`). Map its values onto the kit's colour, radius and type tokens
in one place (declare them unlayered in `web/src/lib/app/app.css`, which wins over the kit's
`albitor-kit` layer); never hardcode a hex in a component. Dark values are derived, not authored: keep
the dark scheme the kit already ships unless the platform hands you derived values.

### Typography

- Inter-led stack (Geist, SF Pro, Segoe UI as fallbacks) with a display stack for headings (`heading_font_family`); semibold 600 headings with -0.02em tracking.
- Scale: body 0.9375rem, secondary 0.8125rem, heading 1.125rem, page title 1.75rem. Keep the title for landing and overview pages; working screens start at the heading size.
- Hierarchy comes from size, weight and spacing. Put ids, branches, hashes and other machine strings in a monospace face.

### Density

- Comfortable. 36px minimum target; generous whitespace between groups, tight spacing inside a group.
- One primary action per view. Lists and tables are airy rather than packed; use progressive disclosure for detail.

### Shape and elevation

- One 10px radius everywhere (controls, cards, menus); 1px hairline borders.
- Elevation is layered: a 1px hairline plus a soft multi-step shadow on raised surfaces and floating layers, lifting floating layers further than page surfaces. No glows, no hard offset shadows, no heavy outlines.

### Colour

- Neutrals are near-achromatic and cool: text `#0d1013`, page `#f7f8fa`, white surfaces. The deep teal accent `#0a7f8f` appears only on actions, marks and links.
- Two border strengths: the subtle `border` hairline for decoration, the stronger `control_border` (3:1) for inputs and outline buttons.
- Status colours always with a glyph and a word; tinted pill fills at about 15% with a 1px border.

The authored light palette: text `#0d1013`, secondary text `#565d68`, border
`#e3e6eb`, control border `#858c9a`, surface `#ffffff`, page
`#f7f8fa`, accent `#0a7f8f` with `#ffffff` on it. The accent is a default:
the app's own brand colour replaces it and the platform recalculates the dependent shades.

## Dark mode

- Cool near-black page, panels one step lighter with a 1px hairline, like Linear, Vercel and Supabase dark themes: dark is the native design, not an inverted light one.
- No shadows in dark; separation is the lighter surface plus the hairline. Controls keep the stronger 3:1 border.
- Accent lifted to 4.5:1 for links and fills, primary stays a solid fill with dark ink, no glow.
- Status colours lightened, always with a glyph and a word. Ink capped near #e1e8f2.

## Accessibility floor

The look never lowers the WCAG 2.2 AA floor: body text 4.5:1, control borders and the focus ring
3:1, a visible focus ring on every interactive element, and status never conveyed by colour alone.
If a brand colour cannot meet the floor as written, shift the shade, do not break the contrast.
