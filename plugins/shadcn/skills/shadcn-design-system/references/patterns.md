Source: shadcn/ui (https://ui.shadcn.com/), Radix UI, and common modern-web-app conventions. MIT-licensed. Distilled reference — verify against the live source for current detail.

# shadcn/ui patterns

Patterns are reusable arrangements of components for recurring product problems. shadcn/ui gives you the building blocks; these patterns show how they compose. They apply to the default archetype for this plugin: a **net-new modern web app** (SME / startup / internal tool) — dashboards, CRUD, settings, auth, onboarding.

Overarching rules:
- Prefer a shadcn/Radix component over hand-rolled markup for every interactive element (the "0 ad-hoc components" bar).
- Keep the happy path keyboard-operable and screen-reader-announced — most of that comes from using the primitives correctly (labels, titles, accessible names).
- Style with tokens; support light and dark from day one.
- Server-validate; client validation (Zod + React Hook Form) is an enhancement, not the security boundary.

## Form patterns

### Single form (settings, create/edit)
Build with the shadcn **Form** component (React Hook Form + Zod). Each field is `FormField` → `FormItem` → `FormLabel` + `FormControl` (wrapping `Input`/`Select`/`Checkbox`/etc.) + `FormDescription` + `FormMessage`. This wires labels, `aria-describedby`, `aria-invalid`, and announced errors for you.
- Do: one clear primary action (`<Button type="submit">`); disable + show a spinner/pending state on submit; keep entered values on validation error; group related fields in a `Card`.
- Don't: use `placeholder` as the label; validate only on the client; auto-submit on field change.

### Validation and errors
Zod schema drives validation; `FormMessage` renders the per-field error linked to the control. For a form-level error, show an `Alert variant="destructive"` at the top and move focus to it. Write errors in plain language that say what's wrong and how to fix it (WCAG 3.3.1/3.3.3).

### Multi-step / wizard
Split a long form into steps with a visible progress indicator (`Progress` or a stepper). Preserve entered data across steps (don't re-ask — WCAG 3.3.7 Redundant Entry). Provide Back/Next as real buttons; validate each step server-side. End with a review step before a consequential submit (3.3.4).

### Asking for common information
Set the right `type`/`inputMode`/`autoComplete`:
- **Name:** single "Full name" `Input` unless you truly need parts; `autoComplete="name"`. Don't assume structure.
- **Email:** `type="email"`, `autoComplete="email"`, `spellCheck={false}`; validate leniently; confirm ownership via a link/code rather than "type it twice".
- **Password:** `type="password"`, `autoComplete="new-password"` / `"current-password"`; allow paste and password managers; state rules up front; consider a show/hide toggle (a `Button size="icon"` with an accessible name). Follow current NCSC/OWASP guidance — no forced arbitrary composition/rotation.
- **Phone:** `type="tel"`, `autoComplete="tel"`, `inputMode="tel"`; accept spaces/formats.
- **Dates:** Date Picker (Calendar + Popover) for browsing/future dates; labelled numeric `Input`s for memorable dates like DOB. `inputMode="numeric"`.
- **Payment:** use a PCI-compliant provider (Stripe Elements etc.) rather than collecting raw card fields; use correct `autoComplete` tokens (`cc-number`, `cc-exp`, `cc-csc`) if you must render fields.
- **Sensitive/equality data:** collect only with a clear need; make optional with a "Prefer not to say"; explain why.

## Page and layout patterns

### App shell
`Sidebar` (collapsible, becomes a `Sheet` on mobile) + a top bar. Wrap nav in `<nav>` landmarks, put a **skip link** to `#main-content` first in the DOM, and mark the primary region with `<main id="main-content">`. Keep nav order consistent across pages (WCAG 3.2.3) and help/account controls in a consistent place (3.2.6).

### Dashboard
A grid of `Card`s (metrics, `Chart`s, recent activity). Every chart needs a text/table alternative — charts are images to assistive tech. Use `Skeleton` for loading and announce load completion. Keep a single logical `<h1>` (page title) and a sensible heading hierarchy for the sections.

### Data table / list view
`Table` for simple data; **Data Table** (TanStack Table) for sort/filter/paginate/select. Header cells are `<th scope>`; provide a caption or a preceding heading. Row actions go in a `DropdownMenu` (keyboard-operable), not click-only icons. Provide filtering via labelled `Input`/`Select`/Combobox above the table. Show an **empty state** when there are no rows (see below).

### Empty states
When a list/table/dashboard has no data yet, show a centred `Card`-like block: a short heading, one line of guidance, and a primary action (`Button`) to create the first item. Don't show a bare empty table.

### Detail / master-detail
List on one side, detail on the other (or a `Sheet`/`Drawer` for the detail on smaller screens). Keep the selected item's state reflected in the URL where possible so it's linkable and back-button-friendly.

### Auth screens (sign in / sign up / reset)
Centred `Card` with a `Form`. Follow the email/password guidance above. Provide a clear route between sign-in and sign-up, "forgot password", and social/SSO buttons (each a real `Button` with an accessible name). Announce success/failure; don't rely on a toast alone for a failed sign-in — render an inline `Alert`.

## Interaction patterns

### Confirming a destructive action
Use **Alert Dialog** (not a plain Dialog, not `window.confirm`). Title states the action ("Delete project?"), description states the consequence ("This can't be undone."), and the confirm button uses `variant="destructive"`. Focus lands on a safe default; `Esc`/cancel dismisses. Never make destructive actions one-click without confirmation or undo.

### Command palette (⌘K)
`CommandDialog` (Command inside a Dialog) for global search/actions. Register a keyboard shortcut but also expose the same actions through visible UI (the shortcut is an enhancement, not the only route). Ensure results are keyboard-navigable (cmdk handles this).

### Notifications / feedback
Transient success → **Sonner** toast (announced via `aria-live`, short-lived). Anything the user must act on or that must persist → inline `Alert` or a Dialog, not a toast. Long operations → `Progress` with an accessible name and an announced completion.

### Selection and menus
Single choice from many → `Select` (short) or Combobox (long/searchable). Actions on an item → `DropdownMenu`. Multi-select filters → Combobox with checkboxes or a Popover of `Checkbox`es. Prefer a Radix menu over a hand-built dropdown so keyboard/typeahead/roles are correct.

### Loading and optimistic UI
Use `Skeleton` placeholders that match the eventual layout; set `aria-busy` on the region and announce when content arrives (4.1.3). For optimistic updates, roll back visibly and announce the failure.

### Async status without a manual refresh
When a resource has an async processing state (e.g. "processing" → "completed"), reflect the outcome on its own — poll or refetch (React Query / SWR `refetchInterval`, or a WebSocket/SSE subscription) and update the row or detail in place. The user should never have to navigate away and back, or hard-refresh, to see an item finish. Announce the transition (`aria-live` / Sonner toast, 4.1.3) and update any status `Badge` accordingly.

## Theming and consistency patterns

- **Dark mode**: theme via CSS variables + a `.dark` class (see `styles.md`); a `next-themes`-style toggle flips the class. Test both themes for contrast — `text-muted-foreground` on tinted surfaces is the usual failure.
- **Consistent components for the same job**: the same action uses the same component and label everywhere (WCAG 3.2.4) — e.g. "delete" is always an `AlertDialog` with a destructive confirm, "add" is always the same primary `Button`.
- **Variants over one-offs**: if you keep writing the same class combination, add a `cva` variant to the component instead of duplicating class strings.
- **Human labels over raw IDs**: show friendly, human-readable names in the UI ("Speaker 1", not `spk_0`); keep the raw identifier for the URL, `key`, or API call, never as the visible label.
- **Keep the document title in sync**: update `document.title` on client-side navigation so the browser tab and history entry reflect the current route or record, not the initial page title.
