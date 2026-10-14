Source: GOV.UK Design System (https://design-system.service.gov.uk/) and GOV.UK Frontend. Crown Copyright, licensed under the Open Government Licence v3.0. Distilled reference — verify against the live source for current detail.

# GOV.UK Design System patterns

Patterns are reusable solutions to recurring design problems, built from components. They fall into three live-site groups: "Pages" (whole-page patterns), "Help users to…" (journey/task patterns), and "Ask users for…" (collecting specific information). Below they are organised as page patterns, journey patterns, and asking-for-information patterns.

Overarching rules:
- One thing per page: ask one question (or one closely related set) per page in a transaction.
- Build with progressive enhancement; every step must work without JavaScript.
- Use plain language; explain why you need information.
- Only collect information you actually need.

## Page patterns

### Question pages
A page that asks the user one question (or a small group of related fields). The question is the page `<h1>` (label/legend as page heading). Include a Back link, not breadcrumbs. Show hint text where helpful. On error, show an Error summary at the top plus inline Error messages.
- Do: make the label/legend the `<h1>`; keep one decision per page; provide a clear single primary action ("Continue").
- Don't: cram multiple unrelated questions onto one page; use placeholder text instead of labels; auto-submit on selection.

### Start pages
The first page of a transactional service. Explains what the service does, how long it takes, what the user needs, and any eligibility. Has a single green "Start now" button linking into the journey. For services also listed on GOV.UK, the start page lives on www.gov.uk.
- Do: set expectations (time, documents needed, cost); one prominent Start button.
- Don't: ask for any information on the start page.

### Confirmation pages
Shown after the user completes a transaction. Uses the green Panel component with a confirming heading and a reference number. Tells the user what happens next and offers what to do now (e.g. print/save, give feedback).
- Do: give a reference number; explain next steps and timescales; include a "What did you think of this service?" feedback link.
- Don't: make this the only record — also email/send confirmation where possible.

### Check answers (check your answers) — under "Help users to… Check answers"
Lets users review and amend their answers before submitting. Built with Summary list; each row has a "Change" link with visually-hidden context naming the field. Sometimes called "check your answers".
- Do: group answers logically (use multiple summary lists / section headings); make Change links return the user to the right question and back; show this before final submission.
- Don't: omit it for anything non-trivial; lose the user's place when they go back to change an answer.

### Cookies page
A dedicated page explaining what cookies the service sets and letting users change their preferences. Pairs with the Cookie banner. List essential vs non-essential cookies; let users opt in/out of non-essential categories.

### Page not found pages (404)
Friendly 404 explaining the page can't be found and offering routes onward (check the address, start again, contact). Keep the standard page furniture (header/footer).

### There is a problem with the service pages (500 / generic error)
Shown when something has gone wrong server-side. Tell the user it's a problem with the service (not them), that their answers haven't been saved if relevant, and how to get help or try again.

### Service unavailable pages
Planned/maintenance downtime. Say the service is unavailable, when it will be back if known, and an alternative way to do the task if one exists.

### Step by step navigation
Guides users through an end-to-end journey that may span multiple services/pages, presented as numbered steps with the tasks in each. Used for cross-service journeys on GOV.UK (e.g. "Learn to drive a car"). Each step can be expanded/collapsed.

## Journey patterns ("Help users to…")

### Check a service is suitable
Help users self-assess eligibility before they start, using question pages. Route ineligible users out early with a clear explanation. Avoid asking for personal data just to check eligibility.

### Complete multiple tasks
For long applications made of several independent tasks. Use the Task list pattern: a list of tasks each with a status Tag ("Completed", "In progress", "Not started", "Cannot start yet"), a count of completed tasks, and a return point users can come back to.
- Do: let users do tasks in any sensible order and save progress; show statuses in text (not colour only).
- Don't: use it for short linear journeys (use sequential question pages instead).
- **Live status without a manual refresh:** where a status changes asynchronously (a task moving to "Completed", a submitted request moving to "Processed"), reflect it on its own — as a progressive enhancement, poll or refetch so the status Tag updates in place rather than leaving the user to re-load the page to find out. Keep the page correct on a normal reload so no-JS users still see the current status, and announce the change (`aria-live` / 4.1.3).

### Confirm an email address
Verify the user controls an email by sending a link/code and having them confirm. Allow re-sending; explain to check spam; let users change the address if wrong.

### Confirm a phone number
Verify a phone number by sending a one-time code (SMS) the user enters. Allow re-sending and changing the number; give a route if they can't receive the code.

### Contact a department or service team
Provide clear contact options (online form, email, phone, post) with expected response times and accessibility alternatives. Don't make phone the only channel.

### Create accounts
Only require an account when there's a genuine need (e.g. to return later, see history). Explain the benefit. Ask for the minimum, and follow the email/phone confirmation patterns. Consider letting users complete a one-off transaction without an account.

### Create a username
Where a username is needed, prefer email address as the identifier. If a separate username is required, give rules up front, validate clearly, and don't impose arbitrary restrictions.

### Navigate a service
Use consistent navigation: GOV.UK header, Service navigation for service-wide links, Back link within linear journeys, Breadcrumbs only for hierarchical/browse structures. Don't mix Back link and Breadcrumbs.
- **Human labels over raw identifiers:** in page content, service navigation, and check-answers rows, show friendly, human-readable names ("Speaker 1", not `spk_0`); keep the raw reference for the URL or back-end call, never as the label the user reads.
- **Keep the page title accurate on every route:** each page needs a unique, descriptive `<title>` (2.4.2), prefixed with "Error: " on a validation error. Where the service enhances navigation client-side, keep `document.title` in sync as the route changes so the browser tab and history reflect the current page, not the one first loaded.

### Start using a service
See Start pages above — set expectations and provide a single Start button.

### Exit a page quickly
For sensitive services where a user may need to leave fast (e.g. risk of an abuser seeing the screen). Use the Exit this page component; also advise on clearing browser history and avoid leaving sensitive data in the URL or page title.

### Recover from validation errors
The standard error pattern: validate on submit (server-side), show an Error summary at the top of the page (focused on load) listing each error linked to its field, plus inline Error messages on each field with `govuk-form-group--error`. Prefix the page `<title>` with "Error: ".
- Do: write errors in plain language saying what's wrong and how to fix it; keep the user's entered answers; order summary errors to match field order.
- Don't: clear the form on error; use colour alone; show errors only inline without a summary.

## Asking users for information ("Ask users for…")

General: ask only what you need, explain why, set the right `autocomplete` and `inputmode`, and make labels page headings on single-question pages.

### Names
Use a single "Full name" field unless you have a specific need to split it. Don't assume name structure (no required middle name, no fixed first/last). `autocomplete="name"` (or `given-name`/`family-name` if split). Allow apostrophes, hyphens, spaces, accents and long names.

### Addresses
Prefer an address lookup (postcode → select address) with a manual-entry fallback. Manual entry: separate fields for address lines, town/city, postcode, with appropriate `autocomplete` tokens (`address-line1`, `address-line2`, `address-level2`, `postal-code`). Validate postcode leniently (allow spacing/case). Provide an international/manual option where relevant.

### Dates
Use the Date input component (three fields: day/month/year) for memorable dates like date of birth — not a date picker. `inputmode="numeric"`. Don't auto-tab between fields. For relative/near dates a different approach may suit. On error, highlight the specific sub-field(s) and name them in the message.

### Email addresses
Single text input, `type="email"`, `autocomplete="email"`, `spellcheck="false"`. Validate format leniently; don't block valid-but-unusual addresses. Confirm ownership via the "Confirm an email address" pattern rather than a "type it twice" field where possible.

### Phone numbers
Single text input, `type="tel"`, `autocomplete="tel"`, `inputmode="tel"`. Accept spaces and various formats; don't impose strict formatting. Explain why you need it and whether it's for voice or SMS. Verify via "Confirm a phone number" if needed.

### National Insurance numbers
Single text input. Explain where to find it (payslip, NI card, personal tax account) and show the format (e.g. "QQ 12 34 56 C"). Accept lower/upper case and spaces; normalise server-side. Only ask if you genuinely need it.

### Passwords
Use the Password input component (show/hide toggle). `autocomplete="new-password"` when creating, `"current-password"` when signing in. Allow paste and password managers; allow long passphrases and all characters. State rules up front. Don't enforce arbitrary composition rules or forced periodic resets; follow current NCSC guidance.

### Bank details
Ask for account holder name, sort code, and account number (and building society roll number where applicable). Sort code as a suitable input; explain where to find each. Consider account-checking services. Handle securely; never log full details unnecessarily.

### Payment card details
Generally use a government-approved payment provider (e.g. GOV.UK Pay) rather than collecting card details directly. If displaying fields, follow PCI requirements and use correct `autocomplete` tokens (`cc-number`, `cc-exp`, `cc-csc`). Don't store raw card data.

### Equality information (equality/diversity, incl. sex and gender)
Ask sensitive equality questions (ethnicity, religion, sexual orientation, sex, gender) only when there is a clear, lawful need; make them optional with a "Prefer not to say" option; explain why you're asking and how data is used. Follow current guidance for asking about sex and gender (e.g. ask separately and only if needed). Place these questions appropriately (often at the end) and never make them block the core task.
