# GOV.UK pattern rubric

Checks for a visual review of a built page against GOV.UK patterns. Patterns only: it says nothing
about colour, icons or art, which this kit keeps plain. Each line is a yes or no; a no is a finding.
The reasons are in `references/patterns.md` and `references/components.md`.

## Every page

- There is exactly one h1, and it says what the page is for.
- The page has the service name and navigation in the shell, and the main content sits in one `main`.
- The primary action is one green button, and its label names the action ("Create report", not
  "Submit").
- Headings follow on in order, with no level skipped.
- Content sits in the two-thirds column, not stretched across the width.

## A flow or a long form

- One thing per page: each question page asks one question (or one closely related set), and the
  question is the h1.
- A long transaction starts from a task list, and a short one runs question page to question page.
- Question pages have a Back link at the top, to the real previous page. The first page and the
  confirmation page do not.
- It has a check-answers page before anything is sent, with a Change link for every answer, and each
  Change link names what it changes.
- It ends on a confirmation page that gives a reference (where there is one) and says what happens
  next.

## Status and messages

- A status is words in a Tag ("In progress", "Not started"), never colour alone, and a task that
  cannot start yet is plain text with no link.
- A validation failure shows an error summary above the h1, each entry linking to its field, and the
  same message beside the field. A notification banner is never used for form errors.
- A notification banner is used sparingly, above the h1, and a success one says what was done.
- Inset text appears once or twice a page, and never carries a warning or an error.

## Facts and lists

- Facts about one record are a summary list (key, value, optional Change), not a table.
- Several like records are summary cards with a heading each, or a table when they are data to
  compare.
- A table is for data: it has a caption and header cells.
