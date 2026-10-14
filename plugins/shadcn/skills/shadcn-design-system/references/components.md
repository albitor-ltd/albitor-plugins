Source: shadcn/ui (https://ui.shadcn.com/docs/components), Radix UI Primitives (https://www.radix-ui.com/primitives), and the libraries shadcn/ui builds on. MIT-licensed. Distilled reference — verify against the live source for current detail (the registry grows and component APIs change).

# shadcn/ui components — the conformance inventory

shadcn/ui components are copied into your project (default path `components/ui/`) by the CLI (`npx shadcn@latest add <name>`). Most interactive ones are styled wrappers over a **Radix UI primitive**, which is where the accessibility behaviour (focus, keyboard, ARIA `role`/`aria-*`/`data-state`) comes from. A few wrap other best-in-class libraries (react-day-picker, embla, cmdk, vaul, react-hook-form, sonner, recharts, input-otp, react-resizable-panels).

**Use this list as the conformance inventory.** The "0 ad-hoc components" bar means: the UI is built from these primitives, not hand-rolled re-implementations of them. If you find yourself writing a `<div role="dialog">`, a bare `<button className="...">`, a custom dropdown, or a from-scratch tabs widget, stop — there is a vetted component here that already handles the accessibility.

General rules that apply to nearly all components:

- Use the component instead of a raw element. A raw `<button>`, `<input>`, `<select>`, `<textarea>`, or custom overlay is an **ad-hoc component** and a conformance failure — it discards the Radix baseline.
- Interactive Radix-based components are **Client Components** (`"use client"` in the Next.js App Router).
- Style via the semantic Tailwind tokens; keep the `focus-visible:ring-*` styles and the Radix `data-state` wiring intact. Don't `outline-none` without an equivalent focus-visible replacement.
- Variants are `cva` props (`variant`, `size`), not ad-hoc classes. Merge overrides with `cn()`.
- `asChild` (Radix Slot) lets a component render as its child element (e.g. a `Button` as a Next.js `<Link>`) while keeping behaviour — use it instead of nesting an `<a>` inside a `<button>`.

## Registry — component → underlying primitive/dep

| Component | Wraps | Interactive? | One-line purpose |
| --- | --- | --- | --- |
| Accordion | `@radix-ui/react-accordion` | Yes | Vertically stacked, expandable sections |
| Alert | (styled markup, `role="alert"`) | No | Callout for important inline information |
| Alert Dialog | `@radix-ui/react-alert-dialog` | Yes | Modal that interrupts for a confirm/cancel (destructive actions) |
| Aspect Ratio | `@radix-ui/react-aspect-ratio` | No | Constrain content to a ratio (images, video) |
| Avatar | `@radix-ui/react-avatar` | No | User image with graceful fallback |
| Badge | (styled markup + `cva`) | No | Small status/label pill |
| Breadcrumb | (styled `<nav>`/`<ol>`) | No | Show location in a hierarchy |
| Button | `@radix-ui/react-slot` (for `asChild`) + `cva` | Yes | Trigger an action; variants + sizes |
| Calendar | `react-day-picker` | Yes | Date-grid calendar |
| Card | (styled markup) | No | Content container (Header/Title/Description/Content/Footer) |
| Carousel | `embla-carousel-react` | Yes | Swipeable/scrollable slides |
| Chart | `recharts` | Yes | Themeable charts wired to the CSS tokens |
| Checkbox | `@radix-ui/react-checkbox` | Yes | Single on/off control |
| Collapsible | `@radix-ui/react-collapsible` | Yes | Toggle a single region open/closed |
| Combobox | `@radix-ui/react-popover` + `cmdk` (Command) | Yes | Autocomplete/searchable select (composed) |
| Command | `cmdk` | Yes | Command palette / fuzzy-search menu |
| Context Menu | `@radix-ui/react-context-menu` | Yes | Right-click menu |
| Data Table | `@tanstack/react-table` + Table | Yes | Sortable/filterable/paginated table (composed) |
| Date Picker | Calendar + `@radix-ui/react-popover` | Yes | Pick a date (composed) |
| Dialog | `@radix-ui/react-dialog` | Yes | Modal window overlaying the page |
| Drawer | `vaul` (built on Radix Dialog) | Yes | Bottom-sheet drawer, mobile-friendly |
| Dropdown Menu | `@radix-ui/react-dropdown-menu` | Yes | Menu of actions from a trigger |
| Form | `react-hook-form` + `zod` (+ Radix Label/Slot) | Yes | Accessible form wiring (label/control/message/aria) |
| Hover Card | `@radix-ui/react-hover-card` | Yes | Preview card on hover (sighted, non-essential) |
| Input | (styled `<input>`) | Yes | Single-line text field |
| Input OTP | `input-otp` | Yes | One-time-passcode entry |
| Label | `@radix-ui/react-label` | No | Accessible label bound to a control |
| Menubar | `@radix-ui/react-menubar` | Yes | Desktop-app-style menu bar |
| Navigation Menu | `@radix-ui/react-navigation-menu` | Yes | Site navigation with dropdowns |
| Pagination | (styled `<nav>` + links) | No | Previous/next/numbered page links |
| Popover | `@radix-ui/react-popover` | Yes | Floating content anchored to a trigger |
| Progress | `@radix-ui/react-progress` | No (status) | Progress bar (`role="progressbar"`) |
| Radio Group | `@radix-ui/react-radio-group` | Yes | Choose exactly one option |
| Resizable | `react-resizable-panels` | Yes | Draggable resizable panels |
| Scroll Area | `@radix-ui/react-scroll-area` | Yes | Custom-styled, accessible scroll container |
| Select | `@radix-ui/react-select` | Yes | Dropdown select with typeahead |
| Separator | `@radix-ui/react-separator` | No | Visual/semantic divider |
| Sheet | `@radix-ui/react-dialog` | Yes | Dialog that slides in from an edge |
| Sidebar | composed (Sheet + buttons + provider) | Yes | Full app sidebar shell |
| Skeleton | (styled placeholder) | No | Loading placeholder |
| Slider | `@radix-ui/react-slider` | Yes | Select a value/range on a track |
| Sonner (Toast) | `sonner` | Yes | Toast notifications (`aria-live`) |
| Switch | `@radix-ui/react-switch` | Yes | On/off toggle |
| Table | (styled `<table>` markup) | No | Semantic tabular data |
| Tabs | `@radix-ui/react-tabs` | Yes | Switch between equivalent panels |
| Textarea | (styled `<textarea>`) | Yes | Multi-line text field |
| Toggle | `@radix-ui/react-toggle` | Yes | Two-state button (pressed/unpressed) |
| Toggle Group | `@radix-ui/react-toggle-group` | Yes | Group of toggles (single/multiple) |
| Tooltip | `@radix-ui/react-tooltip` | Yes | Hint on hover/focus (non-essential info) |

> Note: the older `Toast` (`@radix-ui/react-toast`) is superseded by **Sonner** in current shadcn/ui — prefer Sonner for new work. The registry also continues to grow (e.g. Sidebar, Chart, Input OTP were later additions); confirm the current set at the source.

## Action, form, and input components

### Button — `Button` (Slot + `cva`)
Variants: `default`, `secondary`, `destructive`, `outline`, `ghost`, `link`. Sizes: `default`, `sm`, `lg`, `icon`. Use `variant`/`size` props, not ad-hoc classes. For navigation, render `asChild` around an `<a>`/`<Link>` (`<Button asChild><Link href="…">…</Link></Button>`) so it is a real link. **Icon-only buttons (`size="icon"`) must have an accessible name** — `aria-label` or `sr-only` text; an SVG alone is unnamed.

### Input / Textarea — `Input`, `Textarea`
Styled native `<input>`/`<textarea>`. Always pair with a `Label` (`htmlFor` ↔ `id`) or wrap in `Form`. Set `type`, `inputMode`, `autoComplete` appropriately (`type="email"`, `autoComplete="email"`, `inputMode="numeric"`). Never rely on `placeholder` as a label.

### Label — `Label`
Radix Label; associates text with a control via `htmlFor`. Clicking it focuses/toggles the control. Required for every standalone input/checkbox/switch/radio not already labelled by `Form`.

### Form — `Form` (React Hook Form + Zod)
The accessible way to build forms. `FormField`/`FormItem`/`FormLabel`/`FormControl`/`FormDescription`/`FormMessage` wire up `id`, `htmlFor`, `aria-describedby`, `aria-invalid`, and announced error messages automatically. Use a Zod schema for validation. Prefer this over hand-wiring labels and errors — it satisfies several WCAG criteria (3.3.1/3.3.2/3.3.3, 1.3.1, 4.1.3) by construction.

### Select — `Select` (Radix Select)
Accessible dropdown with full keyboard support and typeahead. Use for a single choice from a moderate list. For long/searchable lists use **Combobox** (Popover + Command). Always give it a `SelectTrigger` with a meaningful accessible name and a `SelectValue` placeholder. For a short set of mutually exclusive options, **Radio Group** is often clearer.

### Radio Group / Checkbox / Switch — `RadioGroup`, `Checkbox`, `Switch`
`RadioGroup` = choose exactly one (wrap items with labels). `Checkbox` = independent on/off (single consent or multi-select). `Switch` = immediate on/off toggle of a setting. Each is a Radix primitive with correct roles/states; each needs an associated `Label`. Don't use a Switch where a form-submit Checkbox is expected, and vice versa.

### Combobox — Popover + Command
Composed, not a single import: a `Popover` containing a `Command` (cmdk) list for searchable/autocomplete selection. Use for long option lists. Ensure the trigger has an accessible name and the selected value is announced.

### Slider — `Slider` (Radix Slider)
Value/range on a track, full keyboard support (arrows/Home/End). Provide an accessible name (`aria-label`) and, where the value isn't otherwise visible, `aria-valuetext`. Not a substitute for a precise numeric input when exact entry matters.

### Input OTP — `InputOtp` (input-otp)
Segmented one-time-passcode field. Keeps a single accessible input under the segmented display; allow paste.

### Calendar / Date Picker — `Calendar` (react-day-picker), Date Picker (Calendar + Popover)
Calendar is a keyboard-navigable date grid. Date Picker composes it inside a Popover triggered by a Button. For a plain memorable date (e.g. date of birth) a set of labelled numeric `Input`s can be more accessible than a calendar — choose by task.

## Overlay and menu components

### Dialog / Alert Dialog — `Dialog`, `AlertDialog` (Radix)
`Dialog` = general modal; `AlertDialog` = interruptive confirm/cancel for consequential/destructive actions (its buttons are the required choice; no dismiss-by-clicking-outside by default). Radix handles focus trap, `Esc` to close, focus return, and `aria-modal`. **Always render `DialogTitle`** (visually hide with `sr-only` if there's no visible title) so the dialog has an accessible name; add `DialogDescription` for context.

### Sheet / Drawer — `Sheet` (Radix Dialog), `Drawer` (vaul)
`Sheet` slides in from an edge (side panels, mobile nav, filters); `Drawer` is a mobile-first bottom sheet. Both inherit Dialog accessibility — same title/description requirement.

### Dropdown Menu / Context Menu / Menubar — Radix menu primitives
Action menus with full keyboard navigation, typeahead, and correct `menu`/`menuitem` roles. `DropdownMenu` opens from a trigger, `ContextMenu` on right-click, `Menubar` is a horizontal desktop-style bar. Use these instead of a hand-built `<ul>`-of-`<div>`s menu. Right-click-only menus need a keyboard/`⋯`-button alternative.

### Navigation Menu — `NavigationMenu` (Radix)
Primary site navigation with accessible dropdown panels. Use for top-level nav; not for in-page action menus (use Dropdown Menu).

### Popover / Hover Card / Tooltip — Radix
`Popover` = click-triggered floating content (can contain interactive elements). `Tooltip` = brief hint on hover/focus for a control — **non-essential, supplementary text only** (hover/focus content must be dismissible/persistent per 1.4.13; don't hide essential info in a tooltip). `HoverCard` = richer hover preview for sighted pointer users — never the only route to information (no keyboard/touch equivalent to hover), so keep the content reachable another way.

### Command — `Command` (cmdk)
Command palette / fuzzy-search menu (⌘K style). Composable list with keyboard navigation. Use inside a `Dialog` for a global palette (`CommandDialog`).

## Display, layout, and feedback components

### Card — `Card`
Container with `CardHeader`/`CardTitle`/`CardDescription`/`CardContent`/`CardFooter`. Use for grouped content; the title should be a real heading where it acts as one in the page outline.

### Table / Data Table — `Table`, Data Table (TanStack Table)
`Table` renders semantic `<table>`/`<thead>`/`<th scope>` markup — use for tabular data, not layout. **Data Table** composes it with `@tanstack/react-table` for sorting/filtering/pagination/row-selection. Keep header cells as `<th scope>`; provide a caption/heading; make row actions real `DropdownMenu`s (keyboard-operable), not click-only icons.

### Tabs — `Tabs` (Radix)
Switch between equivalent panels with correct `tablist`/`tab`/`tabpanel` roles and arrow-key navigation. Don't use for sequential content that must be read in order, or as primary page navigation.

### Accordion / Collapsible — `Accordion`, `Collapsible` (Radix)
`Accordion` = multiple expandable sections (single or multiple open); `Collapsible` = one toggle region. Correct `aria-expanded`/`aria-controls` wiring. Don't hide content most users need behind them.

### Alert / Badge / Progress / Skeleton / Separator / Avatar / Aspect Ratio
- **Alert** — inline callout (`role="alert"` for important ones); `default` and `destructive` variants; convey meaning in text/icon, not colour alone.
- **Badge** — small status label (`cva` variants). Text must carry the status; colour is a secondary cue.
- **Progress** — `role="progressbar"` bar; give it an accessible name and value where meaningful.
- **Skeleton** — loading placeholder; pair with an `aria-busy`/announced loading state so AT users aren't left silent.
- **Separator** — Radix separator; decorative vs semantic (`decorative` prop) matters for AT.
- **Avatar** — image with fallback initials; give informative avatars an accessible name (`alt`/label), decorative ones none.
- **Aspect Ratio** — constrains media to a ratio; still give images meaningful `alt`.

### Sonner (Toast) — `sonner`
Transient notifications rendered in an `aria-live` region so they're announced without stealing focus. Keep messages short; don't put essential, must-act-on information *only* in a toast (it disappears). For confirm/cancel use a Dialog instead.

### Carousel / Resizable / Scroll Area / Sidebar / Chart
- **Carousel** (embla) — provide prev/next buttons (keyboard-operable), don't autoplay without a pause control (2.2.2), and don't hide essential content only in later slides.
- **Resizable** (react-resizable-panels) — draggable panels; ensure a keyboard means to resize or that resizing isn't essential (2.1.1/2.5.7).
- **Scroll Area** (Radix) — styled scroll container that stays keyboard-scrollable.
- **Sidebar** — full app-shell sidebar (collapsible, mobile Sheet). Keep nav landmarks (`<nav>`) and a skip link to main content.
- **Chart** (recharts) — themeable via the CSS tokens; always provide a text/table alternative to the data (charts are images to AT).
