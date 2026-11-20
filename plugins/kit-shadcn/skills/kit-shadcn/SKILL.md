---
name: kit-shadcn
description: How to build pages from the kit-shadcn component kit on a SvelteKit app composed from web-svelte (shadcn-style components on Bits UI, one token set), the design system the app's UI is built and reviewed against. Use when a build adds or changes a page, form, table, dialog, toast, empty state, card, status pill, tabs, chart, stat tile, icon or the navigation under web/src, themes the app, or touches web/src/lib/kit.
metadata:
  capability: design-system
---

# kit-shadcn: the component kit

The kit follows the shadcn ownership model. The components are **source files in your app**
(`web/src/lib/kit/`), seeded once and yours to edit; there is no runtime package to upgrade. The
behaviour that is hard to get right (focus trap, Escape, scroll lock, returning focus) comes from
[Bits UI](https://bits-ui.com), the headless Svelte primitives that shadcn-svelte is built on.
Colours, radius and type come from one token file.

Import from the barrel with a relative path (SvelteKit 3 has no `$lib` alias here):

```ts
import { Button, Field, Table, Dialog, EmptyState, toast } from '../../lib/kit';
import { Card, Pill, StatTile, Tabs, Icon } from '../../lib/kit';
import House from '@lucide/svelte/icons/house';
```

## Rules

1. **Use a kit component if one exists.** A `<div>` modal or a `<span onclick>` button throws away
   the accessibility the kit carries. `references/components.md` lists what exists.
2. **Style with tokens, never hex.** Use `var(--primary)`, `var(--muted-foreground)` and the rest
   (`references/tokens.md`). A hardcoded colour breaks dark mode and the contrast audit.
3. **Label every control.** `Field` ties the label, hint and error to the control; never use a
   placeholder as the label.
4. **Name every icon-only control** with visible text or an `aria-label`. Icons are Lucide
   (`@lucide/svelte`), imported one by one from `@lucide/svelte/icons/<name>` (never the barrel) and
   drawn through `Icon`, which hides a decorative icon from assistive technology. Status is a `Pill`
   (a glyph and a word), never colour alone.
5. **A dialog always has a title and a description.** `Dialog` requires both props.
6. **Values are text.** `Table` cells, toast messages and nav labels render as text. Never feed user
   input to `{@html}`.

## Pages

A page is `web/src/routes/<name>/+page.svelte` (see the `svelte-shell` skill for nav entries). Build
it from the table below and the page templates; the `*Fixture.svelte` components, `PagesShowcase.svelte`,
`ContractProbe.svelte` and `SignInHarness.svelte` in the kit's `__fixtures__/` folders are test fixtures
the kit's tests audit, never a starting point and never imported by a route. Each piece has its own
fixture and test, so an app that does not use a piece deletes the piece, its fixture and its test
together; `web/knip.json` keeps the fixture folders out of the end-of-build prune.

| Need | Use |
|---|---|
| A form field | `<Field label hint error bind:value />`; `multiline` for a textarea; `type` for the input type |
| An action | `<Button variant="primary" \| "secondary" \| "destructive" \| "ghost">`; `type="submit"` in a form |
| Tabular data | `<Table caption columns rows />`; an empty `rows` shows an `EmptyState` |
| Nothing to show | `<EmptyState title description icon>` with an `action` snippet saying what to do next; with `icon` it is the first-run screen |
| Group related content | `<Card title level>` (a surface with its own heading; `tone="second"` for a quiet highlight) |
| A status or count | `<Pill tone="success" \| "warning" \| "danger" \| "info" \| "neutral">`: always a glyph and a word |
| Views of one thing | `<Tabs label tabs bind:value>` with a `panel(value)` snippet (Bits UI Tabs); links between pages stay in the nav |
| A person or team | `<Avatar name src size>`: the photo, or initials, named for the person |
| How far along | `<Progress label value max variant="bar" \| "ring" showValue>` |
| Loading, in the content's shape | `<Skeleton width height circle>` inside an `aria-busy` region that says it is loading |
| A headline figure | `<StatTile label value icon delta meter spark>`: label and value read as a pair; the change is in words |
| A small chart | `<Chart title data kind="bar" \| "sparkline">`: SVG in the tokens plus a table of the same values for assistive technology |
| A filter that is on or off | `<FilterChip bind:pressed>`: a toggle button with `aria-pressed` and a check when on |
| A confirmation or small task | `<Dialog bind:open title description>` with an `actions` snippet |
| Feedback after an action | `toast.push({ message, tone })` (`tone`: `info`, `success`, `error`) |

`AppShell` already renders the one `<Toaster />`; do not render a second. Toasts never disappear on
a timer and each can be dismissed.

Patterns (`references/patterns.md`) cover forms with validation, destructive confirmation, empty
and error states, and navigation on a phone (the shell's Menu button, the tab bar and 4 plus More,
and the drawer; overflow, focus and ARIA).

### Navigation: structure and style are separate

`AppShell` takes the one destinations list (the merged `nav`) and renders it through `NavShell`; the
look's tokens style it, and the structure is chosen by three props, never by editing the shell:

- `layout`: `top` (the default, links in the header row) or `side` (a column beside the page, from 48rem).
- `phoneNav`: the look's stance for a phone. `drawer` (the default) is the Menu button. `tabs` is a
  fixed bottom tab bar for 2 to 5 destinations, or 4 tabs plus a More button that opens the rest as
  a list for 6 or more. One destination is not worth a bar, so `tabs` falls back to `drawer`. Read
  the stance from the look skill's "Phone navigation" section (a look that leans to a tab bar says
  so) and pass it from the root layout. Pick one: a tab bar and a menu button never carry the same
  destinations.
- `icons`: a map from a destination's `href` to a Lucide icon, drawn beside the label. It is keyed
  by href, so a renamed route drops its icon silently: update the map with the route.

`aria-current="page"` is on the current link and never on More (the button says "(current page is in
More)" in words), targets are 44px and the bar pads for the home indicator. `references/patterns.md`
has the rules behind it.

### The account area: Sign out, theme and save status have one home

`AppShell`'s `account` snippet is the header's right-hand end, on desktop and on a phone. Account,
theme and save controls live there and nowhere else: not in a page body, a footer, a settings card
standing in for them or a floating corner. `AccountArea` is the designed content for it:

```svelte
<script lang="ts">
	import { AccountArea, AppShell } from '$lib/kit';
	// the app's own session, theme and save state
</script>

{#snippet account()}
	<AccountArea name={user.name} {theme} onthemechange={setTheme} {saveStatus} onsignout={signOut} />
{/snippet}

<AppShell {nav} {path} {account} title="Example app">{@render children()}</AppShell>
```

Each control draws only when its prop is given, so an app with no theme switch or no autosave shows
neither. `saveStatus` (`idle`, `saving`, `saved`, `error`) is a polite live region and always says
its state in words. The controls are 44px targets; below 48rem the labels and the name collapse to
icons (the accessible names stay) so the header stays one row beside the Menu button. A tab-bar
phone has no Menu button and the account area sits beside the title instead (as the `phoneNav`
setting decides). `accountarea.test.ts` checks it.

## Page templates

Start every new page from the template that matches it. The templates are worked examples, not
seeded files: they sit in `examples/web/src/lib/kit/pages/` beside this skill, laid out by
repository path. Copy the ones the app needs to `web/src/lib/kit/pages/`, each with its test (`listpage.test.ts` for
`ListPage`) and its fixture from `__fixtures__/` (`ListPageFixture.svelte`; they run axe over the
template in every state), plus `pages.test.ts` and the folder's shared `PagesShowcase.svelte` (the
frame), `SignInPageFixture.svelte` and `helpers.ts`. A template the app does not use needs none of its
test or fixture. No route or component imports a fixture, and the fixture check fails if one does.
Import from `'../../lib/kit/pages'`, then edit them: they are the app's code once copied and
never reference this skill. The kit's own pieces (`Button`, `Field`, `Table` and the rest) and
`SignInPage` are already in the repository. Do not rebuild a template by hand.

| The page is | Start from | It gives you |
|---|---|---|
| A collection of records | `ListPage` | Header with actions, a filter group (Fields or `FilterChip`s), `Table`, a first-run `EmptyState` with an icon and a next step (`emptyIcon`), loading skeletons and failed state |
| One record | `DetailPage` | Back link, a status `Pill` beside the title (`badge`), actions, the facts in a `Card` as a description list, related sections, loading and failed states |
| Create or edit one record | `FormPage` | Labelled fields in a `Card`, a focused error summary, a named submit, cancel, a busy guard |
| An overview | `DashboardPage` | `StatTile` figures (icon, change in words, meter or sparkline), sections below (put each in a `Card`), loading skeletons, failed state, and a first-run state (`firstRun`) |
| Preferences, in groups | `SettingsPage` with `SettingsSection` | Each group a `Card` named by its heading, one save, the same error summary |
| Sign in | `SignInPage` | Email and password with the right `autocomplete`, one alert for the failure; presentation only |

### Tables on a phone

`Table` and `ListPage` take `narrow` (measured on the table's own width, under 40rem, not the
viewport). Choose by how much vertical space the list can spend:

- `narrow="stacked"`: each row becomes a card and every field stays visible, labelled by its column.
  Use when each record needs all its fields on screen and a longer scroll is fine (few records, or
  records people read one at a time). Cost: tall rows.
- `narrow="priority"`: the row keeps the primary columns on one or two lines; give the columns that
  can drop `priority: 'secondary'` and they move into a Details disclosure under the first cell. Use
  when the list must stay compact and scannable (many records, one or two columns identify a row).
  The first column is never dropped. Cost: secondary data is one tap away.
- `narrow="scroll"` (default): sideways scroll, nothing changes. Use for wide, comparison-style data
  where column alignment matters more than reading one row.

Both modes are in `__fixtures__/ListPageFixture.svelte` and audited in `listpage.test.ts`. Values stay text.

`PageHeader` (the one `<h1>`, a description, a back link, a status `badge`, actions) and `StateNotice` (loading
`status` with skeletons, failure `alert` with a retry) are the skeleton under every template; use them directly for
a page that fits none of the above, such as a wizard step or an import report.

How to use one:

1. Copy the template's usage comment (the top of the file) into the route and fill in the props. A
   page is still `web/src/routes/<name>/+page.svelte`; the template is a component inside it.
2. Fetch in `+page.ts` or the page, and pass `status` (`loading`, `error`, `ready`) with `rows`,
   `fields` or `stats`. Give `onretry` so a failed load can be tried again.
3. Edit the template when the product needs something it lacks (a column with markup, a second
   action). Keep its headings in order (one `<h1>`, then `<h2>`), and keep every control labelled.
4. A destructive action opens a `Dialog`; it never fires on one click.
5. `SignInPage` is presentation. Call the identity plugin's client from `onsubmit` and pass back a
   plain `error` sentence. If the auth plugin already ships sign-in routes, restyle those with these
   pieces instead of adding a second sign-in route.

## The app's mark

The app's logo is not yours to draw. The platform writes its files beside `look.json` and rewrites
them on every compose: `web/static/brand/` holds `mark.svg` (the one the shell and the sign-in page
draw), `favicon.svg`, `favicon.ico`, `apple-touch-icon.png`, `icon-192.png`, `icon-512.png`, their
dark versions and `manifest.webmanifest`, and `src/app.html` links the icon, the touch icon and the
manifest. The root layout passes `mark="/brand/mark.svg"` to `AppShell`, and `SignInPage` takes the
same optional `mark`. The mark has an empty `alt` (the app's name is beside it) and is not rendered
when its file is missing. **Do not redraw, recolour, replace or inline the mark**, and do not add a
second logo to a page; the files are platform-owned and rewritten from the logo selected at each
build, so never edit them (an edit is overwritten), and a logo change reaches the app on its next
build.

## Theming

Everything reads the custom properties in `tokens.css`, declared in the `albitor-kit` cascade layer so
any unlayered declaration wins. `tokens.css` imports `web/src/lib/look/look.css`, which the platform
writes from the app's look on every build as `--look-*` custom properties; each kit token reads its
`--look-*` property and keeps the kit value as the fallback. So the look reaches the app with no
mapping from you: **never edit `look.css`** (it is overwritten; the kit seeds it empty so the import
resolves when the app has no look) and do not remap tokens by hand. Put the app's own styles in
`web/src/lib/app/app.css`, over the kit's tokens, and leave `tokens.css` and the components alone for a
colour, spacing or shape change. Components use the tokens for spacing (`--space-*`, scaled by the
look's density), row and control height, border width, shadow, heading type, focus width and
motion; use them in your own CSS rather than a rem padding, a `1px` border or a fixed shadow. Dark
mode follows the visitor's preference unless the page sets `data-theme="light"`,
and `data-theme="dark"` forces it.

If you override colours, keep `contrast.test.ts` passing. It computes WCAG 2.2 AA ratios (4.5:1 for
text, 3:1 for the control border, focus ring and primary fill) for every pair the components use,
in light and dark. `kit.test.ts` and the piece tests beside it render the shell and each piece and run axe on them; axe cannot judge
contrast without a layout engine, which is why the contrast test exists.

Accessibility statements and VPAT/ACR (when one is needed, what it contains, a fill-in template) are
in `references/accessibility-statement.md`. The wider WCAG 2.2 AA audit content lives in the
`test-playwright` plugin.

## Accessibility built in

| Piece | What the kit already does |
|---|---|
| `AppShell` | Skip link first, one `main` landmark, a labelled `nav`, `aria-current="page"` on the current link, visible focus |
| `Field` | `label for`, `aria-describedby` to the hint and error, `aria-invalid`, an "Error:" prefix a screen reader reads |
| `Table` | A caption, `scope` on every header, a keyboard-focusable scroll region; `narrow` modes keep the header row available to assistive technology |
| `Dialog` | Named by its title, described by its description, focus trapped and returned (Bits UI) |
| `Toaster` | Always-present `status` (polite) and `alert` (assertive, for errors) regions |
| `NavShell` | One labelled `nav`; `aria-current="page"` on the link, never on More; 44px targets; More says when the current page is in it |
| `Pill`, `StatTile`, `FilterChip` | Status as a glyph and a word; a figure read with its label and its change in words; a toggle with `aria-pressed` and a check |
| `Tabs`, `Avatar`, `Progress` | Bits UI: arrow keys, panel tied to tab; named image; named `progressbar` with its value |
| `Chart`, `Skeleton`, `Icon` | An image with a text table of the same values; hidden from assistive technology; hidden unless given a label |

## Design principles

These carry from the shadcn design system and hold for every screen, kit piece or not. The
detail, and the map from other shadcn components to their Bits UI primitive, is in
`references/design-system.md`.

- **The kit is the inventory.** A screen is built from kit components and Bits UI primitives. A
  hand-rolled dialog, menu, tabs or select is an ad-hoc component and a defect.
- **Semantic tokens only.** Roles (`--primary`, `--muted-foreground`), never a hex or a raw size.
  Every colour is a pair: text and its surface are chosen together.
- **Variants, not one-offs.** A new look is a new `variant` in the component, not a long style
  override at each call site.
- **Never remove the focus ring.** Replace an outline only with an equally visible indicator that
  keeps 3:1 against its neighbours.
- **Colour is never the only signal.** Pair a destructive or success colour with text or an icon.

## Do not

- Do not add Tailwind: this kit styles with scoped CSS over tokens, and the web-svelte build has no
  Tailwind plugin. If you do want it, that is a change to `vite.config.ts` (see `svelte-shell`).
- Do not hand-roll focus management or portals; use Bits UI, which the dialog already wraps.
- Do not remove the focus outlines.
- Do not edit a kit component, the barrel or the tokens from a feature. Features built in parallel
  collide on them at every merge. Wrap or extend a piece in the feature's own directory (a
  `RosterTable.svelte` that renders `Table`); a change every feature needs goes in the shared
  skeleton node, once. Lists features add to are glob-assembled (`svelte-shell`, "Add to a registry").
