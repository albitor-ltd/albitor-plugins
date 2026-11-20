---
name: look-consumer
description: "Friendly, warm and playful: for mobile-first personal and lifestyle web apps. Big thumb-sized targets, rounded type, 24px corners, a warm white page with white cards and a berry accent, a few things at a time."
capability: visual-language
---

# Consumer look

Friendly and warm. Rounded type, 24px corners, 48px targets, warm white neutrals with a berry accent, spacious layouts.

A look is a pure style layer over the app's chosen component kit and markup. It changes tokens
(type, colour, radius, density, elevation), never the accessibility behaviour of a component. The platform paints the preview from `preview-tokens.json` at this plugin's root and
derives the full light and dark scheme from it; the build must use those derived tokens, not
re-invent them.

## Suits

Friendly, phone-first apps: personal, family, community, wellness, hobby, learning and booking tools
used in short sessions, where warmth and ease beat density. Works with any shell and first screen
the app picks. Avoid for data-heavy admin and multi-column tables.

## Apply it

The composed app carries the look's token set as `web/src/lib/look/look.json` (seeded once, a copy
of this plugin's `preview-tokens.json`) and the platform writes `web/src/lib/look/look.css` from
it on every build: the look's colours, radius, spacing, shadow, type and motion as `--look-*`
custom properties, in light and dark. The kit reads them and maps them onto its own tokens, so the
app already looks like this look. **Never edit `look.css`** (the platform overwrites it) and do not
map tokens by hand or redeclare the kit's colour tokens in light and dark selectors. Styles the
app needs beyond the look go in `web/src/lib/app/app.css`, over the kit's tokens; never hardcode a
hex in a component. The kit offers dark and the runner scans both themes.

This look suggests the **Friendly** font look (`default_font_look` in the manifest) when the person
has not picked one; their pick and any policy lock outrank it.
The look also supplies page examples (below) and a `rubric.md` for the visual check.

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

- Warm neutrals: text `#2b160f`, warm white page `#faf8f5`, white cards `#ffffff`. The berry accent `#b0128f` marks the main action and links.
- Decorative borders stay soft; controls use the 3:1 `control_border`.
- Status colours with a glyph and a word. Keep illustrations warm and slightly desaturated.

The authored light palette: text `#2b160f`, secondary text `#6e4533`, border
`#e8dccd`, control border `#a97a50`, surface `#ffffff`, page
`#faf8f5`, accent `#b0128f` with `#ffffff` on it. The accent is a default:
the app's own brand colour replaces it and the platform recalculates the dependent shades.

## Signature

What makes an app in this look recognisable. These are positive recipes; build them in, and review a built page against `rubric.md`.

- **Second colour:** carries small highlights and the data, never the main action: the eyebrow above a focus card, the icon chip on a figure tile, the scattered dots in the art. The berry accent marks the main action and links.
- **Icons:** Lucide through the kit's `Icon`, large (20px) in the nav and on buttons, one beside every nav destination and one on each figure tile.
- **Navigation styling:** the look styles whatever shell the structure picks; it never picks one.
  Top: the destinations in a rounded header row, fully rounded current pill. Side: a roomy column
  with rounded current pills and 48px rows. Tabs: a bottom bar with fully rounded current tabs, the
  main action floating above it. Hub: large soft tiles (24px corners) each with an icon chip. None:
  no chrome, a friendly heading and the one large action.
- **Figures and focus:** wherever a figure or a next step appears it is a friendly tile with an
  icon, and the one thing to do next is one card with a progress ring and a single large action,
  not a grid of equal boxes.
- **Header band:** where a screen opens with a heading, it may use a band tinted from the accent,
  a friendly heading and the look's art, sized down on a phone.
- **Art:** the round, soft `EmptyArt` in the header band and on every first-run page.
- **Motion:** playful but short. A soft lift on press and a smooth ring or bar fill, about 200ms; nothing loops, and none under reduced motion.

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
- **Look-specific:** shapes are round and soft: circles, blobs and 24px-radius cards that overlap,
  one berry mark, a couple of small scattered dots. Warm and slightly desaturated: use the accent at
  reduced opacity for tints, never a second hue, and keep the art about 160px on a phone so the
  sentence and the bottom-reach action stay first.

## Phone navigation

Stance (the fallback when the structure gives no phone navigation): lean strongly towards a bottom
tab bar; this look is mobile-first and the thumb reaches the bottom.

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

### Wire it

The shell is props, not prose to remember. In the root `web/src/routes/+layout.svelte`:

1. Pass `layout` and `phoneNav` to `AppShell` from the structure's shell directive (the app's pick
   owns the shell; see the kit-shadcn SKILL "Navigation: structure and style are separate" for how
   each directive maps to these props, and for `hub` and no shell). Where the build was given no
   directive, fall back to this look's stance above: `phoneNav="tabs"` and `layout="top"`. Never
   choose the shell from the look.
2. Build an `icons` map from the app's own nav, `href` to Lucide component, and pass it too. Import
   each icon on its own from `@lucide/svelte/icons/<name>` (never the barrel), give every
   destination one, and update the map when a route is added, renamed or removed (a missing entry
   shows the label alone).

`examples/web/src/routes/+layout.svelte` beside this skill shows the whole layout wired this way.
Keep the app's own providers, title and mark; add only `layout`, `phoneNav` and `icons`.

## Page examples

Styled examples of every kit page template are in `examples/web/src/lib/look/pages/`, laid out by
repository path: Overview (a tracker's home: header band, focus card with a progress ring, figure tiles and a floating action; use it only when the structure names a dashboard or a tracker-collection), Habits (list that stacks into cards on a phone), Habit (detail with a progress ring), NewHabit (form), Preferences (settings), FirstRun and Setup (one question at a time with big choices). `pages.test.ts` renders each page (loading, failed and
empty states included) and runs axe, one `h1` and heading-order checks over it.

They are worked examples, not seeded files. Copy the ones the app needs, with `pages.test.ts`
trimmed to match, to `web/src/lib/look/pages/`, then edit them: replace the sample data and copy,
fetch in `+page.ts` or the page, and pass `status`. They import the kit's page templates
(`DashboardPage`, `ListPage`, `DetailPage`, `FormPage`, `SettingsPage`, `HomePage`,
`CataloguePage`, `ItemPage`, `ArticlePage`, `CollectionPage`, `HubPage`, `StepPage`, `PageHeader`,
`StateNotice` and their small pieces), which come from the `kit-shadcn` skill's examples, so copy those first. Once
copied they are the app's code and never reference this skill. Do not rebuild a page from scratch
when one of these fits.

### Non-CRUD example app

For a **`catalogue-directory`** or **`booking`** structure (top bar, phone tabs: `layout="top"`,
`phoneNav="tabs"`) start from the non-CRUD example app, in the same folder: Classes (search and
filter soft cards, on the kit's `CataloguePage`), Class (the facts and a large "Book a place" card,
on `ItemPage`) and Bookings (the person's own bookings as a grid with the look's `EmptyArt` on
first run, on `CollectionPage`). `pages.test.ts` covers each state. Copy the kit's page templates
first; no imagery is used.

## Reviewing a page

`rubric.md` beside this file is a short checklist for a visual review of a built page against this
look (the heading font loaded, the page colour from `look.css`, the second colour in the data,
icons in the nav, a designed first-run state, density). Use it on screenshots or a rendered page.

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
