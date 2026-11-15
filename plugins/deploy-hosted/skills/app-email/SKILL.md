---
name: app-email
description: Use when an app composed with the deploy-hosted plugin sends email of its own - an order receipt, a reminder, a digest, a notice to a member - or the plan's Emails step declared app-sent templates (services.callable has an email entry). Send a declared template through the gateway with the email example (copied to api/internal/email), and give the app a Notifications page under Settings when any template is optional or consent mail. Never write an SMTP or SES client, a sender, a subject or a body in the app.
metadata:
  capability: deploy-target
---

# App-sent email (deploy-hosted)

A hosted app sends no mail itself. The owner confirmed each email's wording, reason and basis in
the platform's Emails step; the platform holds the text, renders it in a fixed shell with a footer
that says who sent it and why, and sends it from its own mail domain. The app names a template and
supplies its values. Copy the example only when `albitor.yaml` has an `email` entry under
`services.callable`; an app without one carries none of it.

## Starting points

From the app's repository root (module path `webapp/...`; change the import if the module is named
otherwise):

```sh
cp -r <skill dir>/examples/api/internal/email api/internal/email
cp <skill dir>/examples/api/internal/app/notifications*.go api/internal/app/   # optional or consent mail only
cp -r <skill dir>/examples/web/src/lib/notifications web/src/lib/notifications # same
cp -r "<skill dir>/examples/web/src/routes/(app)/settings" "web/src/routes/(app)/settings"
```

The email example needs the gateway client (`api/internal/gateway`, see the `gateway-services`
skill), and the Notifications API needs the `auth-hosted` identity package. Then edit the copies:
they are the app's code from then on, tests included.

## Send a declared template

The template ids are exactly the ones in the manifest, and nothing else is accepted:

```yaml
services:
  auth: { ... }               # the email service mails the app's own members, so sign-in is required
  callable:
    - service: email
      templates: [order-receipt, weekly-digest]
```

```go
mail, err := email.New(svc, []string{"order-receipt", "weekly-digest"}, logger) // once, in main
...
res, err := mail.Send(ctx, "order-receipt", order.Email, email.Vars{
	"order": email.Number("1042"),                                  // typed: number
	"name":  email.Text(order.CustomerName),                        // typed: one line, no link
	"track": email.Link("https://"+host, "/orders/1042"),            // typed: a link on the app's own host
}, email.ForEvent("order-1042-receipt"))
```

- **The values are the template's placeholders, exactly.** The Emails step lists each template's
  variables and their types (`text`, `number`, `link`); a missing, extra or wrongly typed one is
  `400 invalid_variable`. At most 12 values, each at most 500 characters. A `text` value is one line
  with no URL; a `link` is `https` on the app's own host and nothing else. The constructors check
  this before the call, so a mistake fails in a test.
- **Never write wording in the app.** There is no subject, body, sender, reply-to or link-host
  field, and the gateway refuses one (`400 bad_request`). To change what a mail says, the owner
  edits the template in the app's email settings.
- **`ForEvent` makes a handler safe to run twice.** The same event, template, address and values
  returns the first answer and sends nothing. Name the business event (`order-1042-receipt`), not a
  timestamp. Leave it off for a mail that may truly repeat; Send then uses a fresh key.
- **Never put an address in a log, a URL or an error shown to a person.**

### What each answer means

`Send` returns a `Result` or an error. None of these is retried:

| Answer | What it means | The app does |
| --- | --- | --- |
| `Disposition: sent` | A member got the mail | Nothing more |
| `Disposition: invited` | The person is not a member yet, so the platform's fixed invitation went instead | Nothing more; mail reaches them once they join |
| `Disposition: suppressed` | **Not an error.** The person turned this optional mail off, never agreed to this consent mail, the owner has not confirmed the template yet, or it is members-only and they are not a member. An unconfirmed template is reported this way, with no separate signal | Log it at info (`Send` does) and carry on. Never send again, never "fix" it by switching a preference on for them |
| `404 unknown_template` | Declared, but the platform holds no copy yet | `email.Classify` says skip: log once, carry on |
| `403 not_declared` | The id is not in the manifest's `templates`: a build bug | Skip, and fix the manifest or the code. `Send` already refuses an id outside the list given to `email.New` |
| `409 never_email`, `409 invite_window`, `429 daily_cap`, `403 tenant_paused`, `402` | This mail is not going out now | Skip. Do not retry and do not queue it for later: a loop spends the app's limits and can mail someone twice |
| `400`, `5xx`, a network failure | A request shape bug or an outage | A fault: log the error text and correlation id, and answer an opaque 500 only if the mail was the point of the request |

A mail is a side effect, not the request. Send it after the work it reports has succeeded, and a
failure to send must not undo it or turn a successful request into an error; log the outcome and
return.

## Optional and consent mail: the Notifications page

Each template carries a basis the owner chose:

- **Transactional, not optional**: the person cannot turn it off (a receipt for something they
  bought). It is not listed on the Notifications page.
- **Optional**: on by default; one click in the footer, or a switch on the page, turns it off.
- **Consent**: off until the person agrees. The gateway refuses to send it without an opt-in on
  record (`suppressed`), so an app that never offers the choice sends nothing.

If any template is optional or consent, the app **must** have a Notifications page at
`/settings/notifications`, linked from Settings and from the nav. The example is that page: a switch
per email, the reason beside it, optional mail on and consent mail off to start, and a save that
sends only what the person changed. It calls the app's `GET`/`PUT /api/notifications`, which call
the gateway's `GetEmailPreferences` and `SetEmailPreferences` for the signed-in person's verified
address.

- **Write a `Label` for every template** (`name` and `reason`, in `api/internal/app` where
  `Notifications` is constructed). The reason is the platform's reason sentence for that email, which
  the gateway does not return; copy it from the plan's email list. A template with no label still
  shows, with a generic reason, and is logged, so a missing label is visible and never a hidden
  switch.
- **An opt-in is the person's act.** `SetPreferences` tells the gateway the person did it
  (`ConsentSource: user_action`), which it requires to switch consent mail on. Call it only from a
  request the person made, never from a default, an import or a seed script.
- **Sign-up opt-in.** The hosted sign-in page is the platform's and cannot carry a box of the app's,
  so for an app with consent mail put `OptInPrompt` on the home page: it asks once after first
  sign-in, unticked, and closing it records nothing. An app that builds its own sign-up form may
  also offer an unticked box there and call `PUT /api/notifications` after the account exists.
- Keep the route, the link to it from `settings/+page.svelte` (or its `nav.ts`) and the
  `data-testid` values. The build's handover check fails an app that declares optional or consent
  templates with no `web/src/routes/**/settings/notifications/+page.svelte` reachable from Settings.
- **Never name the platform** in the page, a label or a reason. People using the app know the
  organisation that runs it, not the platform that built it.

## Tests

The app's tests do not call the gateway. `email.Sender` takes a `Caller` (the gateway client's
`Call`), so fake it as the example's tests do, and cover the paths the app owns: a send that is
`suppressed` still lets the request succeed, a refusal is logged and not retried, and a mail is
sent once when the handler runs twice (`ForEvent`). The Notifications API depends on a small
`PreferenceStore` interface for the same reason.
