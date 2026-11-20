---
name: look-startup
description: "Crisp, quiet and fast: for SaaS and developer-tool products. One confident teal accent on near-neutral surfaces, generous whitespace, a strong type hierarchy with tight display tracking, hairlines and a soft layered shadow."
capability: visual-language
---

# Startup look

Crisp and quiet. Near-achromatic neutrals, one teal accent, 10px corners, hairlines plus a soft layered shadow, hierarchy from type not decoration.

A look is a pure style layer over the app's chosen component kit and markup. It changes tokens
(type, colour, radius, density, elevation), never the accessibility behaviour of a component. The platform paints the preview from `preview-tokens.json` at this plugin's root and
derives the full light and dark scheme from it; the build must use those derived tokens, not
re-invent them.

## Suits

Apps with a modern product feel: SaaS and developer tools, workspaces, planners, trackers,
directories, catalogues, tools for clubs and teams, and onboarding, for people who use the app often
and judge it on speed and clarity. Works with any shell and first screen the app picks. Avoid for
dense back-office tables (use Business) or mobile-first lifestyle apps (use Consumer).

## Apply it

The composed app carries the look's token set as `web/src/lib/look/look.json` (seeded once, a copy
of this plugin's `preview-tokens.json`) and the platform writes `web/src/lib/look/look.css` from
it on every build: the look's colours, radius, spacing, shadow, type and motion as `--look-*`
custom properties, in light and dark. The kit reads them and maps them onto its own tokens, so the
app already looks like this look. **Never edit `look.css`** (the platform overwrites it) and do not
map tokens by hand or redeclare the kit's colour tokens in light and dark selectors. Styles the
app needs beyond the look go in `web/src/lib/app/app.css`, over the kit's tokens; never hardcode a
hex in a component. The kit offers dark and the runner scans both themes.

This look suggests the **Geometric** font look (`default_font_look` in the manifest) when the person
has not picked one; their pick and any policy lock outrank it.
The look also supplies page examples (below) and a `rubric.md` for the visual check.

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

- Neutrals are near-achromatic and cool: text `#0d1013`, page `#f7f8fa`, white surfaces. The deep teal accent `#08737f` appears only on actions, marks and links.
- Two border strengths: the subtle `border` hairline for decoration, the stronger `control_border` (3:1) for inputs and outline buttons.
- Status colours always with a glyph and a word; tinted pill fills at about 15% with a 1px border.

The authored light palette: text `#0d1013`, secondary text `#565d68`, border
`#e3e6eb`, control border `#858c9a`, surface `#ffffff`, page
`#f7f8fa`, accent `#08737f` with `#ffffff` on it. The accent is a default:
the app's own brand colour replaces it and the platform recalculates the dependent shades.

## Signature

What makes an app in this look recognisable. These are positive recipes; build them in, and review a built page against `rubric.md`.

- **Second colour:** carries the data and quiet highlights, never an action: a meter fill, a sparkline, the icon chip on a stat tile, the tinted current item in the nav, `Card tone="second"` and the `info` Pill. The teal accent stays on actions, marks and links.
- **Icons:** Lucide through the kit's `Icon`, one beside every nav destination and one on each stat tile; 16px in buttons, 18px in the nav. Icons name things, they never carry the only cue.
- **Navigation styling:** the look styles whatever shell the structure picks; it never picks one.
  Side: a hairline-edged column with the tinted current item in the second colour, 18px icons.
  Top: a bar with a 1px bottom hairline and the current link tinted the same way. Tabs: a hairline
  top border, the teal accent on the current tab. Hub: each tile is a layered card (hairline plus
  a soft shadow) with an icon chip in the second colour. None: no chrome, the page title and its
  one primary action only.
- **Figures:** wherever a figure appears (a stat, a hub tile, a card), it is layered (hairline
  plus a soft shadow) with an icon, a change in words, and a meter for a share or a sparkline for
  a trend. Never a bare number in a box.
- **Page header:** the page title with one primary action beside it; everything else in the page is
  secondary.
- **Art:** the layered-cards `EmptyArt` on a first-run page, with the second colour as a thin trace.
- **Motion:** subtle. Short ease on hover, focus, press and meter fills only; none for decoration, and none under reduced motion.

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
- **Look-specific:** shapes are precise layers: stacked cards at the look's 10px radius with 1px
  hairline edges and one accent mark, no glow, no gradient, no offset shadow. Keep it about 152px
  and understated; a technical product empty state reads better with less.

## Phone navigation

Stance (the fallback when the structure gives no phone navigation): lean towards a bottom tab bar
when the product has 2 to 5 core destinations, and a menu button and drawer when it has more.

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
- Use a hairline top border (a border token) as the separator; the teal accent marks the current tab.

### Wire it

The shell is props, not prose to remember. In the root `web/src/routes/+layout.svelte`:

1. Pass `layout` and `phoneNav` to `AppShell` from the structure's shell directive (the app's pick
   owns the shell; see the kit-shadcn SKILL "Navigation: structure and style are separate" for how
   each directive maps to these props, and for `hub` and no shell). Where the build was given no
   directive, fall back to this look's stance above: `phoneNav="tabs"` and `layout="side"`. Never
   choose the shell from the look.
2. Build an `icons` map from the app's own nav, `href` to Lucide component, and pass it too. Import
   each icon on its own from `@lucide/svelte/icons/<name>` (never the barrel), give every
   destination one, and update the map when a route is added, renamed or removed (a missing entry
   shows the label alone).

`examples/web/src/routes/+layout.svelte` beside this skill shows the whole layout wired this way.
Keep the app's own providers, title and mark; add only `layout`, `phoneNav` and `icons`.

## Page examples

Styled examples of every kit page template are in `examples/web/src/lib/look/pages/`, laid out by
repository path: Overview (layered tiles, a bar chart and a card in the second colour), Projects (list with a search field and a filter chip), Project (detail with status, a sparkline and tabs), NewProject (form), Preferences (settings), FirstRun and Setup (a step flow with a progress bar). `pages.test.ts` renders each page (loading, failed and
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

- Cool near-black page, panels one step lighter with a 1px hairline, like Linear, Vercel and Supabase dark themes: dark is the native design, not an inverted light one.
- No shadows in dark; separation is the lighter surface plus the hairline. Controls keep the stronger 3:1 border.
- Accent lifted to 4.5:1 for links and fills, primary stays a solid fill with dark ink, no glow.
- Status colours lightened, always with a glyph and a word. Ink capped near #e1e8f2.
- Dark accent. The light accent fails 4.5:1 on the dark page, so on dark pages (`#09090b` and the
  panel steps above it) use `#2fb5c6` for links, accent text and the primary fill (dark ink `#09090b`
  on it); it holds 4.5:1 or better on `#09090b`, `#18181b` and `#27272a`.

## Accessibility floor

The look never lowers the WCAG 2.2 AA floor: body text 4.5:1, control borders and the focus ring
3:1, a visible focus ring on every interactive element, and status never conveyed by colour alone.
If a brand colour cannot meet the floor as written, shift the shade, do not break the contrast.
