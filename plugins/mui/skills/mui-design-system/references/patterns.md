Source: Material UI (MUI) documentation (https://mui.com/) and Material Design (https://m3.material.io/). MUI is MIT-licensed; Material Design is © Google. Distilled reference — verify against the live source for current detail.

# Material patterns

Patterns are **framework-agnostic** reusable arrangements of Material components that solve recurring screen-level problems in a consumer or general-purpose app. Build them from the vetted Material components (see `components.md`) rather than bespoke markup, and style them with theme tokens (see `styles.md`). The component names below (`TextField`, `Dialog`, `AppBar`, …) are the Material components each library provides — the same pattern composes from SMUI, m3-svelte, or `@mui/material` under its own import paths (see `frontend-conventions.md`). A couple of code-idiom notes (e.g. `sx`, `document.title` on React-router navigation) are React examples; the equivalent applies in Svelte.

Overarching rules:
- Compose from components; don't hand-roll controls a component already provides.
- Keep it responsive: lay out with `Grid`/`Stack`/`Container` and `useMediaQuery`, not fixed pixel widths.
- Every interactive element is keyboard-operable and labelled; status is never colour-only.
- Theme once, reuse everywhere — palette, type, spacing, shape come from the theme.

## App shell and navigation

### App bar + drawer shell
The standard desktop app frame: a fixed `AppBar` (menu `IconButton` with `aria-label`, `Typography` title, right-aligned actions/avatar) over a `Drawer` of navigation links.
- Desktop: `Drawer variant="permanent"` as a side rail; content in a `Box`/`Container` offset by the AppBar/drawer width.
- Mobile: `Drawer variant="temporary"` toggled by the menu button (focus-trapped overlay), or a `BottomNavigation` for top-level destinations.
- Put nav links in a `List` of `ListItemButton`s inside a `<nav>` landmark; mark the current item. Ensure the fixed AppBar doesn't obscure focused controls (2.4.11).

### Breadcrumbs and page hierarchy
Use `Breadcrumbs` with real `Link`s to show where a page sits; mark the current page as plain text. Don't combine breadcrumbs with a redundant "back" control that does the same thing.
- **Human labels over raw IDs:** in breadcrumbs, tables, and detail headers show friendly, human-readable names ("Speaker 1", not `spk_0`); keep the raw identifier for the route param or API call, never as the visible label.
- **Keep the document title in sync:** update `document.title` on client-side navigation (React Router / TanStack Router) so the browser tab and history reflect the current page or record, not the initial app title.

### Tabs vs stepper
`Tabs` switch between equivalent views of the same screen. `Stepper` is for a sequential multi-step flow (e.g. checkout) — show step status in text, keep steps reachable, and validate each step before advancing.

## Forms

### Basic form
A `Box component="form"` (or `<form>`) with `TextField`s, each with a `label` and, where useful, `helperText`. Lay fields out with `Stack spacing={2}` or `Grid`. One primary `Button variant="contained" type="submit"`; secondary actions are `outlined`/`text`.
- Label every field; never rely on `placeholder` as the label.
- Choose the right control: `RadioGroup` for a few visible options, `Select`/`Autocomplete` for many, `Checkbox` for on/off/multi, `Switch` for an immediate setting, date via the MUI X pickers.
- Set `type`/`autoComplete`/`inputMode` (`type="email"`, `autoComplete="email"`, `inputMode="numeric"`) so browsers autofill and mobile keyboards adapt (WCAG 1.3.5).

### Validation and errors
Validate on submit (and optionally on blur). For each invalid field set `error` and put the message in `helperText` (text, not colour alone — 1.4.1/3.3.1). Summarise errors at the top for long forms, move focus to the first error or the summary, and keep the user's entered values. Say what's wrong and how to fix it (3.3.3).

### Multi-step form
Wrap steps in a `Stepper`; keep each step's fields in their own panel; persist entered data across steps (don't re-ask — 3.3.7); add a review step before a consequential submit (3.3.4). Every step must be operable by keyboard.

## Data and content

### Data table
For simple data use `Table` with proper `TableHead`/`th scope` headers and a caption. For large or interactive data (sort, filter, paginate, select, edit, virtualise) use the MUI X `DataGrid`. Add a toolbar (`Toolbar`/`Box`) above the table for search and bulk actions; right-align numeric columns. Provide pagination via `TablePagination` or the grid's built-in paging.

### List / feed
`List` of `ListItemButton`s (interactive) or `ListItem`s (static), with `ListItemAvatar`/`ListItemIcon` + `ListItemText` and optional secondary actions. Use `Divider` between groups and subheaders for sections.

### Cards / grid of items
Lay `Card`s out in a responsive `Grid`. Each card: `CardMedia` (meaningful image needs `alt`), `CardContent`, `CardActions`. For a whole-card click target use `CardActionArea` (a real button/link).

### Dashboard
A `Grid` of `Card`/`Paper` tiles: KPI stats (`Typography` + supporting `Chip`/trend), charts (MUI X `Charts` with a text/data-table alternative), and a recent-activity `List` or `DataGrid`. Keep tiles reflowing to one column on small screens.

## Feedback and overlays

### Confirmation dialog
`Dialog` with `DialogTitle` + `DialogContentText` + `DialogActions` (a cancel `text` button and a confirm `contained` button; use `color="error"` for destructive confirms, paired with clear text). Set `aria-labelledby`/`aria-describedby`. Let Esc/backdrop cancel.

### Toasts and inline messages
Transient success/info → `Snackbar` (auto-dismiss, short). Persistent, in-context status → `Alert` with the right `severity` (icon + text, not colour-only). Critical, blocking errors → an `Alert` or `Dialog` the user must acknowledge, not a Snackbar.

### Loading, empty, and error states
- **Loading:** `Skeleton` shaped like the incoming content (calmer than a spinner) or `CircularProgress`/`LinearProgress` for actions; use `Backdrop` for a blocking load.
- **Empty:** an explanatory `Typography` + an icon + a primary action ("Create your first…"), not a blank screen.
- **Error:** an `Alert severity="error"` (or an error page) that says what happened and offers a retry; don't lose the user's input.
- **Async status without a manual refresh:** a resource in an async processing state ("processing" → "completed") should update itself — poll/refetch (or subscribe over WebSocket/SSE) so the `DataGrid`, list, or detail reflects completion in place, without the user leaving and returning or reloading. Announce the change (a `Snackbar` or an updated status `Chip`, 4.1.3).

## Account and settings

### Sign in / sign up
A centred `Container maxWidth="xs"` with a `Card`/`Paper`, a titled form of labelled `TextField`s (`type="email"`, `type="password"` with `autoComplete="current-password"`/`"new-password"`), a primary submit `Button`, and secondary `Link`s (forgot password, switch to sign up). Allow paste and password managers; don't impose arbitrary composition rules; offer non-puzzle auth where possible (3.3.8).

### Settings page
Group settings into `Card`/`Paper` sections with headings; use `Switch` for immediate toggles, `RadioGroup`/`Select` for choices, and `TextField` for values. Persist immediately (with a `Snackbar` confirmation) or provide an explicit Save; be consistent about which.

### Profile / account menu
An avatar `IconButton` (with `aria-label`) in the AppBar opening a `Menu` of account actions. Keep the menu keyboard-operable and labelled.

## Theming patterns

### Light/dark mode
Drive mode from `palette.mode`; offer a toggle (persist the choice). Use `theme.palette.*` tokens everywhere so both modes work without per-component overrides. Respect `prefers-color-scheme` for the initial value.

### Branding
Set your brand colours in the theme `palette` (primary/secondary and, if needed, custom tokens), your fonts in `typography.fontFamily`, and shape/spacing to taste. Don't fork components to restyle — theme them. Keep contrast ≥ 4.5:1 for text after rebranding (1.4.3).
