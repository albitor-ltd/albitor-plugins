---
name: look-consumer
description: "Friendly, warm and playful: for mobile-first personal and lifestyle web apps. Big thumb-sized targets, rounded type, 24px corners, a warm cream palette with a berry accent, a few things at a time."
capability: visual-language
---

# Consumer look

Friendly and warm. Rounded type, 24px corners, 48px targets, cream neutrals with a berry accent, spacious layouts.

A look is a pure style layer over the app's chosen component kit and markup. It changes tokens
(type, colour, radius, density, elevation), never the structure or the accessibility behaviour of a
component. The platform paints the preview from `preview-tokens.json` at this plugin's root and
derives the full light and dark scheme from it; the build must use those derived tokens, not
re-invent them.

## Suits

Mobile-first personal and lifestyle apps: habit, planning, family, wellness and hobby tools used in short sessions on a phone, where warmth and ease beat density. Avoid for data-heavy admin and multi-column tables.

## Apply it

The composed app carries the look's token set as `web/src/lib/look/look.json` (seeded once, a copy
of this plugin's `preview-tokens.json`). Map its values onto the kit's colour, radius and type tokens
in one place (declare them unlayered in `web/src/lib/app/app.css`, which wins over the kit's
`albitor-kit` layer); never hardcode a hex in a component. An unlayered declaration beats the kit's
layered dark block too, so a token you override in the light selector (`:root`) silently stays light
on the dark page. **Override tokens in both the light and dark selectors, or not at all.** Declare
the dark values in `:root:not([data-theme="light"])` (inside `@media (prefers-color-scheme: dark)`)
and in `:root[data-theme="dark"]`. Use the platform's derived `dark_*` tokens when it hands you
them; otherwise use the dark accent below. The kit offers dark and the runner scans both themes.

### Typography

- Rounded system stack (`ui-rounded`, SF Pro Rounded, Nunito and similar); heavy 800 headings for a friendly voice, with comfortable body text.
- Scale is large: body 1.0625rem, secondary 0.9375rem, heading 1.5rem, page title 2.5rem. On narrow phones let the title clamp down so it never wraps beyond two lines.
- Sentence-case, plain language, short labels; avoid all-caps.

### Density

- Spacious. 48px minimum target and generous padding; show a few things at a time and put the single main action within thumb reach at the bottom.
- Prefer one column, large cards and bottom sheets to tables and side panels.

### Shape and elevation

- 24px radius on cards and sheets; controls are fully rounded within it; 2px borders; 3px focus ring.
- Elevation is raised: a lighter or warmer surface step with a soft shadow. Keep it friendly, not ornamental: no hard outlines, offset shadows, clashing colours or novelty typography.

### Colour

- Warm neutrals: text `#2b160f`, cream page `#fff7ee`, peach surfaces `#ffe8d1`. The berry accent `#b0128f` marks the main action and links.
- Decorative borders stay soft; controls use the 3:1 `control_border`.
- Status colours with a glyph and a word. Keep illustrations warm and slightly desaturated.

The authored light palette: text `#2b160f`, secondary text `#6e4533`, border
`#f2c9a0`, control border `#a97a50`, surface `#ffe8d1`, page
`#fff7ee`, accent `#b0128f` with `#ffffff` on it. The accent is a default:
the app's own brand colour replaces it and the platform recalculates the dependent shades.

## Phone navigation

Stance: lean strongly towards a bottom tab bar; this look is mobile-first and the thumb
reaches the bottom.

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
- Use the 48px target, fully rounded tab pills and the berry accent for the current tab, with a
  visible label.

## Dark mode

- Warm neutral, not brown: saturation capped, page about 8% lightness, surface lifted.
- Elevation by lighter surface; no drop shadows. Decorative borders stay subtle, controls use the 3:1 control border.
- Accent and status held to about 5:1 on the surface; label on accent is the dark page ink.
- Illustrations about 15% less saturated, no pure white; true black only as an OLED opt-in.
- Dark accent. The light accent fails 4.5:1 on the dark page, so on dark pages (`#09090b` and the
  panel steps above it) use `#f06cd0` for links, accent text and the primary fill (dark ink `#09090b`
  on it); it holds 4.5:1 or better on `#09090b`, `#18181b` and `#27272a`.

## Accessibility floor

The look never lowers the WCAG 2.2 AA floor: body text 4.5:1, control borders and the focus ring
3:1, a visible focus ring on every interactive element, and status never conveyed by colour alone.
If a brand colour cannot meet the floor as written, shift the shade, do not break the contrast.
