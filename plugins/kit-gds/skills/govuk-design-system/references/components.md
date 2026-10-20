Source: GOV.UK Design System (https://design-system.service.gov.uk/) and GOV.UK Frontend. Crown Copyright, licensed under the Open Government Licence v3.0. Distilled reference — verify against the live source for current detail.

# GOV.UK Design System components

There are 34 components in the GOV.UK Design System. Each is available as a Nunjucks macro (imported from `govuk-frontend`) and as plain HTML you can copy. Macro names follow the pattern `govuk<ComponentName>` (e.g. `govukButton`).

General rules that apply to nearly all components:

- Use the component's own classes; do not restyle (no recolouring buttons, no changing input border thickness, no reassigning colour meanings).
- Form components must have a programmatically associated `<label>` (or `<legend>` in a `<fieldset>` for grouped inputs). Never rely on placeholder text as a label.
- JavaScript-enhanced components carry a `data-module` attribute and must be initialised (see `initAll` in frontend-conventions.md). They are built progressively — the non-JS HTML must still be usable.
- Error states use `govuk-error-message` plus a red border/`...--error` modifier on the field, and the field is wrapped in `govuk-form-group--error`.

## Quick reference table

| Component | Macro | Has JS? | One-line purpose |
| --- | --- | --- | --- |
| Accordion | `govukAccordion` | Yes | Show/hide sections of related content |
| Back link | `govukBackLink` | No | Link back to the previous page |
| Breadcrumbs | `govukBreadcrumbs` | No | Show the user's location in a hierarchy |
| Button | `govukButton` | Yes (debounce) | Trigger an action |
| Character count | `govukCharacterCount` | Yes | Textarea with a live remaining-characters count |
| Checkboxes | `govukCheckboxes` | Yes (conditional reveals) | Let users select one or more options |
| Cookie banner | `govukCookieBanner` | No (you wire consent) | Tell users about cookies; capture consent |
| Date input | `govukDateInput` | No | Three text fields for day/month/year |
| Details | `govukDetails` | No (native `<details>`) | Expandable extra detail |
| Error message | `govukErrorMessage` | No | Inline error text on a field |
| Error summary | `govukErrorSummary` | Yes (focus on load) | Summary of all errors at top of page |
| Exit this page | `govukExitThisPage` | Yes | Quickly leave a sensitive page |
| Fieldset | `govukFieldset` | No | Group related form controls with a legend |
| File upload | `govukFileUpload` | Yes (enhanced) | Upload a file |
| GOV.UK footer | `govukFooter` | No | Standard site footer with OGL/licence links |
| GOV.UK header | `govukHeader` | Yes (mobile menu) | Standard GOV.UK black header |
| Inset text | `govukInsetText` | No | Set apart a block of text |
| Notification banner | `govukNotificationBanner` | Yes (focus if important) | Important/success message |
| Pagination | `govukPagination` | No | Previous/next or numbered page navigation |
| Panel | `govukPanel` | No | Confirmation panel (big green box) |
| Password input | `govukPasswordInput` | Yes | Password field with show/hide toggle |
| Phase banner | `govukPhaseBanner` | No | Alpha/Beta phase tag and feedback link |
| Radios | `govukRadios` | Yes (conditional reveals) | Select exactly one option |
| Select | `govukSelect` | No | Dropdown of options |
| Service navigation | `govukServiceNavigation` | Yes (mobile menu) | Service name + navigation links |
| Skip link | `govukSkipLink` | Yes (focus target) | Skip to main content |
| Summary list | `govukSummaryList` | No | Key/value summary, optionally with Change links |
| Table | `govukTable` | No | Tabular data |
| Tabs | `govukTabs` | Yes | Switch between equivalent sections (large screens) |
| Tag | `govukTag` | No | Status label |
| Task list | `govukTaskList` | No | List of tasks with statuses |
| Text input | `govukInput` | No | Single-line text field |
| Textarea | `govukTextarea` | No | Multi-line text field |
| Warning text | `govukWarningText` | No | Highlight important warning with `!` icon |

## Navigation and page-furniture components

### GOV.UK header — `govukHeader`
The black GOV.UK-branded header (crown logo, "GOV.UK"). Use at the very top of every page. Has a JS-enhanced mobile navigation toggle. Do not put service-specific navigation here — use Service navigation for that. For services **on GOV.UK only**: off GOV.UK the crown logo and the "GOV.UK" name must not be used — build your own header using GDS layout and typography with your own brand.

### Service navigation — `govukServiceNavigation`
Shows the service name and (optionally) navigation links beneath the GOV.UK header. Replaces the older pattern of putting the service name in the header. Has a JS mobile menu. Use it for navigation that applies across the whole service.

### GOV.UK footer — `govukFooter`
Standard footer. Includes the Open Government Licence statement and Crown copyright by default — keep these. Add your service's links (accessibility statement, cookies, privacy, terms). For services **on GOV.UK only**: off GOV.UK the OGL and Crown copyright do not apply — build your own footer using GDS layout and typography with your own brand and licensing.

### Phase banner — `govukPhaseBanner`
Shows the service phase (Alpha / Beta) as a Tag plus a sentence with a feedback link. Use while a service is not yet Live. Remove it (or the banner) once Live.

### Breadcrumbs — `govukBreadcrumbs`
Shows where the page sits in a hierarchy and lets users move up levels. Use on hierarchical/browsable sites. Do not use inside a linear transaction — use Back link instead. Gotcha: don't use Breadcrumbs and Back link together.

### Back link — `govukBackLink`
A single "Back" link to the previous step. Use in linear journeys (question pages). Place it at the top of the page, above `<main>`. Make it go to the actual previous page, and ensure it still works without JS (point `href` at a real URL, don't rely on `history.back()` alone).

### Skip link — `govukSkipLink`
"Skip to main content" link, first focusable element on the page. Its target (e.g. `id="main-content"`) must exist and be focusable. JS moves focus to the target on activation. Required for keyboard accessibility — include on every page.

### Pagination — `govukPagination`
Navigate paginated lists or long content split across pages (Previous/Next, optionally numbered). Use real `href`s so it works without JS.

## Form components

All form fields should sit in a `govuk-form-group`. Labels/legends should be headings on single-question pages (see patterns.md). Add hint text with the field's `hint` option; it is linked via `aria-describedby`.

### Text input — `govukInput`
Single-line text. Set `inputmode`/`type`/`autocomplete`/`spellcheck` appropriately (e.g. `autocomplete="email"`, `inputmode="numeric"`). Size with width override classes (`govuk-input--width-10`, `govuk-!-width-one-half`) rather than guessing — size hints the expected input length. Supports prefix/suffix (e.g. £, kg).

### Textarea — `govukTextarea`
Multi-line text. Set `rows` to hint expected length. For a hard/soft limit, use Character count instead.

### Character count — `govukCharacterCount`
Textarea plus a live count of characters or words remaining (`maxlength` or `maxwords`, optional `threshold` to only show near the limit). JS-enhanced; without JS it degrades to a plain textarea with the limit stated in hint text. The live count uses an ARIA live region — do not remove it.

### Select — `govukSelect`
Native dropdown. Use sparingly — radios are usually more accessible for short lists, and an autocomplete or text input is often better for long lists (avoid very long selects). Always have a sensible default/first option and a real `<label>`.

### Radios — `govukRadios`
Choose exactly one option. Wrap in a Fieldset with a legend. Supports a divider ("or") and "conditional reveal" content that appears when an option is selected (JS-enhanced; the revealed content must remain reachable without JS). Use `govuk-radios--inline` only for two short options; stack otherwise. For yes/no use two radios, not a checkbox.

### Checkboxes — `govukCheckboxes`
Select none, one, or many. Wrap in a Fieldset with a legend. Supports an exclusive "or none of these" option (`behaviour: "exclusive"`) and conditional reveals (JS). Use a single checkbox only for a clear binary consent (e.g. "I agree"). Order options logically, not alphabetically unless that aids scanning.

### Date input — `govukDateInput`
Three separate text inputs (day, month, year), not a date picker. Use for memorable dates (date of birth). Each field gets `inputmode="numeric"`. For dates the user must look up (e.g. from a document) consider asking differently. Error handling: highlight the specific field(s) at fault and reference them in the error message. Do not auto-advance focus between fields.

### File upload — `govukFileUpload`
Native `<input type="file">`, optionally JS-enhanced with a styled drop zone and clearer button text. State accepted file types and max size in hint text. Validate type/size server-side; surface errors via the normal error pattern.

### Fieldset — `govukFieldset`
Groups related controls (radios, checkboxes, date inputs) under a `<legend>`. Required so screen readers announce the group question. On a single-question page, make the legend an `<h1>` via `isPageHeading: true`.

### Password input — `govukPasswordInput`
Password field with a JS show/hide toggle and accessible status announcements. Set `autocomplete="current-password"` or `"new-password"`. Without JS it is a normal masked password field. Don't disable paste.

## Feedback, error and status components

### Error summary — `govukErrorSummary`
A box at the top of the page listing every error, each linking to the offending field. Required whenever a page has validation errors. JS moves focus to the summary on page load. Title is usually "There is a problem". Order the errors to match the order of fields on the page. Pair with inline Error message on each field.

### Error message — `govukErrorMessage`
Red inline message attached to a field. Add `govuk-form-group--error` to the group and link the message via `aria-describedby`. Prefix is visually hidden "Error:". Message should say what went wrong and how to fix it, in plain language.

### Notification banner — `govukNotificationBanner`
Important information or a success confirmation (`type: "success"` turns it green). Use sparingly. The success variant and important variant receive focus on load via JS. Don't use it for validation errors (use Error summary) — and don't overuse it or users will ignore it.

### Panel — `govukPanel`
Large green confirmation panel, used on confirmation pages with a heading and (e.g.) a reference number. Put the most important info (reference number) in the `html`/body.

### Tag — `govukTag`
Small coloured status label (e.g. "Completed", "In progress"). Colour variants exist (`govuk-tag--blue`, `--green`, `--grey`, `--red`, etc.) — choose by meaning and ensure the text alone conveys status (colour is not the only signal). Keep text short.

### Warning text — `govukWarningText`
Bold warning with a circled `!` icon and visually-hidden "Warning" text. Use for genuinely important consequences (e.g. legal/penalty), not general emphasis.

### Inset text — `govukInsetText`
Indented block with a left border, to set apart content such as a quote, example, or supplementary note. Don't overuse; not for warnings (use Warning text).

## Disclosure and layout components

### Accordion — `govukAccordion`
Lets users show/hide multiple sections of related content, with a show/hide-all control. JS-enhanced; without JS all sections are open. Use only for long pages of largely independent content the user won't all need. Don't use for content users should read in full or in sequence.

### Details — `govukDetails`
Native `<details>`/`<summary>` expandable disclosure for a single chunk of supplementary content. Lightweight alternative to Accordion for one item. Don't hide content most users need.

### Tabs — `govukTabs`
Switch between sections that are equivalent alternatives (e.g. different regions' data). JS-enhanced on large screens; on small screens / no JS it falls back to stacked headings + content (all visible). Don't use to hide content users need to read sequentially, and don't use for primary navigation.

### Summary list — `govukSummaryList`
Key/value pairs, typically a "check your answers" summary. Each row can have an action ("Change") with visually-hidden text naming what it changes (e.g. "Change your name"). Use `govuk-summary-list--no-border` for a borderless variant. Can be grouped into cards with `govukSummaryList` card options.

### Table — `govukTable`
Tabular data with proper `<th scope>` row/column headers and an optional caption. Use real tables only for data, not layout. Right-align numeric columns with `govuk-table__cell--numeric` and use tabular figures.

### Task list — `govukTaskList`
A list of tasks each with a status Tag (e.g. "Completed", "Cannot start yet"). Used for the "Complete multiple tasks" pattern. Each task title links to the task; status conveyed in text, not colour alone.

## Sensitive-content component

### Exit this page — `govukExitThisPage`
A prominent button (and Esc-key/triple-shift shortcut) to rapidly leave a page and clear it from history, for users who may be in danger (e.g. domestic abuse services). JS-enhanced. Place near the top. Also advise users about browser history. Without JS the button still links away.

## Cookies component

### Cookie banner — `govukCookieBanner`
Tells users about cookies and captures accept/reject before setting non-essential cookies. You must wire up the consent logic and not set non-essential cookies until consent is given. Show a confirmation message after the choice, with a "hide" action. Pair with a dedicated Cookies page (see patterns.md).
