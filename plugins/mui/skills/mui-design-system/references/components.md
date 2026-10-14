Source: Material UI (MUI) documentation (https://mui.com/) and Material Design (https://m3.material.io/). MUI is MIT-licensed; Material Design is © Google. Distilled reference — verify against the live component API pages for current props and behaviour.

# Material component inventory

This is the **conformance inventory** for Material: when a vetted Material component exists for a need, use it rather than hand-rolling markup (that is the basis of a "no ad-hoc components" check). The inventory is shown in the **`@mui/material` (React)** form — package names, tags, and import paths below are the React library. **The same Material components exist in the Svelte libraries** (SMUI: one package per component, e.g. `@smui/button`, `@smui/textfield`; m3-svelte: named imports from `m3-svelte`) — map the component name, keep the same accessibility requirement. See `frontend-conventions.md` for the per-framework import paths and the SMUI variant mapping. In the React form, components come from `@mui/material` unless a package is noted; icons from `@mui/icons-material`; the **MUI X** packages (`@mui/x-*`) are separate installs.

General rules that apply to nearly all components (any library):

- Import from the library, don't reinvent: e.g. `import Button from '@mui/material/Button'` (React) / `import Button from '@smui/button'` (SMUI). Capitalised tags (`<Button>`, `<TextField>`) are Material components; lowercase tags (`<button>`, `<input>`) are raw HTML and usually mean a control that should be a Material component.
- Style via the `sx` prop or `styled()` using **theme tokens** (`theme.palette.*`, `theme.spacing`, `theme.typography.*`), never hardcoded hex/px — so light/dark mode and the scale hold.
- Form controls need an accessible label: `TextField label=…`, or a `FormLabel`/`InputLabel`; icon-only controls (`IconButton`, `Fab`) need `aria-label`.
- Don't convey status by colour alone — pair a colour (`color="error"`) with text or an icon.

## Quick reference table

| Component | Package | Category | One-line purpose |
| --- | --- | --- | --- |
| Autocomplete | `@mui/material` | Input | Text input with a suggestions/typeahead list |
| Button | `@mui/material` | Input | Trigger an action (contained / outlined / text) |
| Button Group | `@mui/material` | Input | Group related buttons |
| Checkbox | `@mui/material` | Input | Toggle one or more options on/off |
| Fab (Floating Action Button) | `@mui/material` | Input | Primary promoted action, floating |
| Radio Group | `@mui/material` | Input | Choose exactly one of a few visible options |
| Rating | `@mui/material` | Input | Star (or custom) rating input/display |
| Select | `@mui/material` | Input | Dropdown of options (`MenuItem`s) |
| Slider | `@mui/material` | Input | Pick a value/range along a track |
| Switch | `@mui/material` | Input | On/off toggle (binary setting) |
| Text Field | `@mui/material` | Input | Single/multi-line text entry (label + helper + error) |
| Toggle Button | `@mui/material` | Input | Grouped on/off selection (e.g. text alignment) |
| Transfer List | `@mui/material` | Input | Move items between two lists |
| Avatar | `@mui/material` | Data display | User/entity image, initials, or icon |
| Badge | `@mui/material` | Data display | Small count/dot overlay on an element |
| Chip | `@mui/material` | Data display | Compact tag / filter / input token |
| Divider | `@mui/material` | Data display | Thin rule separating content |
| Icon / SvgIcon | `@mui/material` + `@mui/icons-material` | Data display | Material icons |
| List | `@mui/material` | Data display | Vertical list of items (interactive or static) |
| Table | `@mui/material` | Data display | Tabular data (`TableHead`/`Body`/`Row`/`Cell`) |
| Tooltip | `@mui/material` | Data display | Hover/focus label for a control |
| Typography | `@mui/material` | Data display | Text at a Material type-scale variant |
| Alert | `@mui/material` | Feedback | Inline success/info/warning/error message |
| Backdrop | `@mui/material` | Feedback | Dim overlay behind a loading/modal state |
| Dialog | `@mui/material` | Feedback | Modal dialog (confirm, form, alert) |
| Progress (Circular/Linear) | `@mui/material` | Feedback | Loading indicator |
| Skeleton | `@mui/material` | Feedback | Placeholder shimmer while content loads |
| Snackbar | `@mui/material` | Feedback | Brief, auto-dismissing toast message |
| Accordion | `@mui/material` | Surface | Expand/collapse sections |
| App Bar | `@mui/material` | Surface | Top (or bottom) application bar / toolbar |
| Card | `@mui/material` | Surface | Contained block of related content + actions |
| Paper | `@mui/material` | Surface | Elevated surface (the base for many components) |
| Bottom Navigation | `@mui/material` | Navigation | Bottom tab bar (mobile primary nav) |
| Breadcrumbs | `@mui/material` | Navigation | Location within a hierarchy |
| Drawer | `@mui/material` | Navigation | Side (or temporary) navigation panel |
| Link | `@mui/material` | Navigation | Styled, accessible navigation link |
| Menu | `@mui/material` | Navigation | Popup list of choices/actions |
| Pagination | `@mui/material` | Navigation | Page through long lists |
| Speed Dial | `@mui/material` | Navigation | Fab that fans out to related actions |
| Stepper | `@mui/material` | Navigation | Multi-step progress (horizontal/vertical) |
| Tabs | `@mui/material` | Navigation | Switch between equivalent views |
| Box | `@mui/material` | Layout | Generic styled container with the `sx` prop |
| Container | `@mui/material` | Layout | Centre content at a max width |
| Grid | `@mui/material` | Layout | Responsive 12-column layout |
| Stack | `@mui/material` | Layout | One-dimensional flex layout with `spacing` |
| Image List | `@mui/material` | Layout | Grid of images (masonry/quilted/woven) |
| Modal / Popover / Popper | `@mui/material` | Utils | Low-level overlays underneath Dialog/Menu/Tooltip |
| Click-Away Listener / Portal / No SSR | `@mui/material` | Utils | Behaviour helpers |
| CssBaseline | `@mui/material` | Utils | Normalise + apply theme baseline (render once) |
| Textarea Autosize | `@mui/material` | Utils | Auto-growing textarea |
| useMediaQuery | `@mui/material` | Utils | Responsive JS breakpoint hook |
| Data Grid | `@mui/x-data-grid` | MUI X | Feature-rich data table (sort/filter/paginate/edit) |
| Date / Time Pickers | `@mui/x-date-pickers` | MUI X | Accessible date, time, and range pickers |
| Charts | `@mui/x-charts` | MUI X | Line/bar/pie/scatter charts |
| Tree View | `@mui/x-tree-view` | MUI X | Hierarchical, expandable tree |

## Inputs

### Button — `<Button>`
Actions. Three variants: `variant="contained"` (high emphasis / primary action), `"outlined"` (medium), `"text"` (low). `color` picks a palette entry (`primary`, `secondary`, `error`…). Add `startIcon`/`endIcon`. Use `<Button component={RouterLink} href=…>` for navigation that looks like a button. Don't use a `<Button>` where a plain link is semantically right — use `<Link>`.

### Icon Button — `<IconButton>`
A button showing only an icon (e.g. close, menu, more). **Must** have an `aria-label` — it has no visible text, so without one it has no accessible name (WCAG 4.1.2). Ensure the 24×24px target-size minimum (2.5.8); the default `size` meets it.

### Fab (Floating Action Button) — `<Fab>`
A circular, elevated button for the single most important action on a screen. Icon-only Fabs need an `aria-label`; the `extended` variant shows text too.

### Button Group / Toggle Button — `<ButtonGroup>` / `<ToggleButtonGroup>`
`ButtonGroup` visually joins related buttons. `ToggleButtonGroup` (+ `ToggleButton`) is for a small set of mutually-exclusive or multi-select toggles (e.g. text alignment); set `exclusive` for single-select and give the group an `aria-label`.

### Text Field — `<TextField>`
The workhorse text input. Always give it a `label` (renders an associated `<label>`); use `helperText` for hints, and `error` + `helperText` for validation messages. `multiline` (with `rows`/`maxRows`) makes it a textarea; `select` turns it into a dropdown. `type`/`autoComplete`/`inputMode` should match the data (`type="email"`, `autoComplete="email"`). Composed of `FormControl` + `InputLabel` + `Input`/`OutlinedInput` + `FormHelperText` if you need finer control.

### Select — `<Select>`
A dropdown, normally inside a `FormControl` with an `InputLabel` (linked via `labelId`), containing `<MenuItem>`s. Use for a moderate list of options; for many options or free-text search use `Autocomplete`; for a few visible options prefer `RadioGroup`. `multiple` enables multi-select (often rendered with `Chip`s).

### Autocomplete — `<Autocomplete>`
Text input with a filtered suggestion list; supports free solo, multiple (tags), async options, and grouping. Renders its input with a `TextField`. Prefer over a long `Select`. Keep the `renderInput` `TextField` labelled.

### Checkbox / Radio Group / Switch — `<Checkbox>` / `<RadioGroup>` / `<Switch>`
Wrap each in a `FormControlLabel` to get a clickable, associated label. `RadioGroup` (+ `FormLabel` in a `FormControl`) is exactly-one selection from a few visible options. `Checkbox` is none/one/many. `Switch` is an immediate on/off setting (like a toggle in settings), not a form submission choice. For a group, use a `FormLabel`/`FormGroup` so the question is announced.

### Slider — `<Slider>`
Pick a value or range on a track. Provide an `aria-label`/`aria-labelledby` and `getAriaValueText`; it is keyboard-operable by default. Not ideal where a precise typed value matters — offer a `TextField` alternative for precision.

### Rating — `<Rating>`
Star (or custom icon) rating, for input or read-only display (`readOnly`). Give it a `name` and an accessible label.

### Transfer List
A composed pattern (two `List`s + move buttons) rather than a single component — see `patterns.md`.

## Data display

### Typography — `<Typography>`
Renders text at a Material type-scale variant (`h1`–`h6`, `subtitle1/2`, `body1/2`, `button`, `caption`, `overline`). Set the semantic element with `component` (e.g. `variant="h1" component="h2"`) so the visual size and the heading level can differ — keep a single logical `<h1>` and correct heading order regardless of visual size.

### List — `<List>`
Vertical list of `ListItem`s. For an **interactive** row (navigates or acts on click) use `<ListItemButton>` (a real button/link) — not a `ListItem` with an `onClick`, which isn't keyboard-operable. Compose with `ListItemIcon`, `ListItemAvatar`, `ListItemText`, `ListItemSecondaryAction`.

### Table — `<Table>`
Tabular data: `TableContainer` › `Table` › `TableHead`/`TableBody` › `TableRow` › `TableCell`. Use `TableCell component="th"` with scope for header cells and a `caption` where helpful. Use a real table for data, not for layout. For sorting, filtering, pagination, virtualisation or editing at scale, use the MUI X **Data Grid** instead.

### Chip — `<Chip>`
Compact element for tags, filters, or input tokens. A deletable chip needs a `deleteIcon` and (for icon-only delete) an accessible label; a clickable chip should behave like a button. Don't rely on colour alone to convey a chip's meaning.

### Avatar / Badge — `<Avatar>` / `<Badge>`
`Avatar` shows an image (needs `alt`), initials, or an icon. `Badge` overlays a small count or dot on its child; make the meaning available in text for assistive tech (e.g. `badgeContent={4}` with context, not just a red dot).

### Tooltip — `<Tooltip>`
Hover/focus label for a control. It must be reachable on focus (not hover only) to satisfy 1.4.13; the wrapped control still needs its own accessible name — a tooltip is not a substitute for an `aria-label`.

### Icon / SvgIcon — `<Icon>` / icons from `@mui/icons-material`
Material icons as React components (`import DeleteIcon from '@mui/icons-material/Delete'`). Decorative icons should be hidden from AT (`aria-hidden`); an icon that conveys meaning on its own needs a text alternative (usually via the surrounding control's `aria-label`).

### Divider — `<Divider>`
A thin rule between list items or sections. Presentational; use headings/landmarks for actual structure.

## Feedback

### Alert — `<Alert>`
Inline coloured message: `severity="success" | "info" | "warning" | "error"`. Carries an icon plus text, so meaning isn't colour-only. Use `AlertTitle` for a heading. For a message that must be announced when it appears, put it in a live region (or use `Snackbar`).

### Snackbar — `<Snackbar>`
Brief, auto-dismissing toast at the edge of the screen, optionally wrapping an `Alert`. Use for transient confirmations ("Saved"), not for critical errors the user must act on. It uses an ARIA live region so it is announced — keep messages short.

### Dialog — `<Dialog>`
Modal dialog for confirmations, forms, or alerts. Compose `DialogTitle` + `DialogContent` (+ `DialogContentText`) + `DialogActions`. Set `aria-labelledby` (to the title id) and `aria-describedby` (to the description id). MUI traps and restores focus and closes on Esc/backdrop by default — keep those. Don't nest dialogs; don't use a dialog where an inline `Alert` would do.

### Progress — `<CircularProgress>` / `<LinearProgress>`
Loading indicators. Use `determinate` with a `value` when progress is known, `indeterminate` when not. Give a busy indicator an accessible label; for content still loading, `Skeleton` is often calmer than a spinner.

### Skeleton / Backdrop — `<Skeleton>` / `<Backdrop>`
`Skeleton` is a placeholder shimmer sized like the incoming content. `Backdrop` dims the screen behind a blocking load or a modal.

## Surfaces

### App Bar — `<AppBar>`
The top application bar (`position="fixed" | "sticky" | "static"`), usually containing a `Toolbar` with a menu `IconButton`, a `Typography` title, and actions. Wrap the app in a `<header>`/landmark sense; ensure a fixed AppBar doesn't obscure focused content (2.4.11) — use `scroll-padding`/offsets.

### Card — `<Card>`
A contained surface for related content: `CardHeader`, `CardMedia` (needs `alt`/`title` for meaningful images), `CardContent`, `CardActions`. For a fully-clickable card use `CardActionArea` (a real button/link) so it's keyboard-operable.

### Paper — `<Paper>`
The base elevated surface (`elevation={0..24}` or `variant="outlined"`). Many components build on it. Elevation must not be the only signal of grouping — keep structure semantic.

### Accordion — `<Accordion>`
`AccordionSummary` (the toggle, with an `expandIcon`) + `AccordionDetails`. Keyboard-operable and correctly ARIA-wired by MUI. Use for independent, optional sections; don't hide content users need to read in sequence.

## Navigation

### Drawer — `<Drawer>`
A side panel for navigation. `variant="permanent"` (desktop rail), `"persistent"`, or `"temporary"` (mobile overlay, focus-trapped). Put nav links in a `List` of `ListItemButton`s inside a `<nav>` landmark.

### Tabs — `<Tabs>`
Switch between equivalent views (`Tabs` + `Tab`, each linked to a tab panel). MUI wires `role="tablist"`/`tab`/`tabpanel` and keyboard arrow navigation — keep the `aria-controls`/`id` associations. Don't use tabs for sequential steps (use `Stepper`) or as primary site navigation.

### Menu — `<Menu>`
A popover list of `MenuItem` actions, opened from a button/`IconButton` (which needs an `aria-label` if icon-only). Focus is managed and it closes on Esc/away. Use for contextual actions, not primary navigation.

### Bottom Navigation — `<BottomNavigation>`
Bottom tab bar for the top-level destinations on mobile (3–5 items with icons + labels). Keep labels visible for clarity; each action is a real button.

### Breadcrumbs / Pagination / Stepper — `<Breadcrumbs>` / `<Pagination>` / `<Stepper>`
`Breadcrumbs` show hierarchy (use real `Link`s; mark the current page). `Pagination` moves through long lists (renders accessible page buttons; give it a nav label). `Stepper` shows progress through a multi-step flow (horizontal/vertical, linear/non-linear) — convey step status in text, not colour alone.

### Link — `<Link>`
A styled, accessible anchor. Use for navigation (real `href`), not for actions (use `Button`). Make link text meaningful out of context ("View invoice", not "click here"); mark "opens in new tab" in the text.

### Speed Dial — `<SpeedDial>`
A `Fab` that fans out to related actions on hover/focus. Give the dial and each action an `aria-label`; ensure keyboard operability (it is by default).

## Layout

### Box / Stack / Container / Grid — `<Box>` / `<Stack>` / `<Container>` / `<Grid>`
`Box` is a styled `div` with the `sx` prop — the general building block. `Stack` lays children in one dimension with `spacing` and `direction`. `Container` centres content at a responsive `maxWidth`. `Grid` is the responsive 12-column system (`<Grid container>` + `<Grid>` with size props); use it for two-dimensional layouts. Prefer these over hand-written fl/grid CSS so spacing tracks the theme. (Note: the Grid API changed across MUI versions — confirm the size-prop syntax for the version in use.)

### Image List — `<ImageList>`
A grid of images (`standard`, `quilted`, `masonry`, `woven`). Each `ImageListItem` image needs meaningful `alt` (or empty `alt` if decorative).

## Utils and low-level overlays

- **`CssBaseline`** — render once near the root to normalise browser styles and apply the theme's background/text. Prefer over a hand-written CSS reset.
- **`Modal` / `Popover` / `Popper`** — the primitives under `Dialog`/`Menu`/`Tooltip`. Reach for the high-level component first; drop to these only for custom overlays, and then re-implement focus management and ARIA yourself.
- **`ClickAwayListener`, `Portal`, `NoSsr`, `TextareaAutosize`, `useMediaQuery`** — behaviour helpers. `useMediaQuery(theme.breakpoints.up('md'))` gives responsive logic in JS.

## MUI X (separate packages, notable capabilities)

- **Data Grid — `@mui/x-data-grid`** (`DataGrid`, and `DataGridPro`/`Premium` commercially): sorting, filtering, pagination, column resizing/pinning, row selection, inline editing, virtualisation. Use instead of a hand-built `Table` when the data set is large or interactive. It ships extensive keyboard support — keep column headers labelled.
- **Date & Time Pickers — `@mui/x-date-pickers`**: `DatePicker`, `TimePicker`, `DateTimePicker`, and range pickers, with an accessible text-field entry plus calendar/clock. Needs a date-library adapter (`AdapterDayjs`/`AdapterDateFns`) inside a `LocalizationProvider`. Prefer these over a raw `<input type="date">` when you need consistent cross-browser behaviour and labelling.
- **Charts — `@mui/x-charts`**: `LineChart`, `BarChart`, `PieChart`, `ScatterChart`, sparklines. Provide text alternatives / a data table for charts so information isn't visual-only (WCAG 1.1.1).
- **Tree View — `@mui/x-tree-view`**: `SimpleTreeView`/`RichTreeView` for hierarchical, expandable data with correct `tree`/`treeitem` ARIA and keyboard navigation.
