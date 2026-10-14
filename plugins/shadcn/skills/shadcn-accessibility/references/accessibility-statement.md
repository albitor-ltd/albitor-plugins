# Accessibility statements and VPAT/ACR

> **Sources:** [W3C WAI: Accessibility Statements generator & guidance](https://www.w3.org/WAI/planning/statements/), the [ITI VPAT / Accessibility Conformance Report (ACR)](https://www.itic.org/policy/accessibility/vpat), and the [European Accessibility Act](https://ec.europa.eu/social/main.jsp?catId=1202) overview.
> Distilled reference — verify against the live source for current detail.

## Do you need one?

A commercial or internal SME web app is usually **not** legally required to publish a UK-public-sector-style prescribed statement. But a clear, honest accessibility statement is good practice, increasingly expected, and sometimes required:

- **W3C recommends** every site publish an accessibility statement — it signals commitment, tells users how to get help, and documents known issues.
- **EU Accessibility Act (EAA)** — from **28 June 2025**, many private-sector products and e-commerce/banking/transport/communication services sold in the EU must be accessible (harmonised to EN 301 549, which references WCAG 2.2 AA) and provide accessibility information. If you sell into the EU, this may apply.
- **US ADA** — courts treat business websites as places of public accommodation; WCAG 2.x AA is the de-facto benchmark in settlements. No prescribed statement, but a public commitment + remediation record helps.
- **US Section 508 / procurement** — selling to US federal (and many enterprise) buyers needs a **VPAT** producing an **Accessibility Conformance Report (ACR)** — a criterion-by-criterion conformance table.

When unsure which regime applies, ask the user about their market (EU / US / UK / enterprise procurement) before choosing the framing.

## What a good statement contains

1. **Commitment + scope** — what product/site it covers and your accessibility goal.
2. **Conformance status** — tied to a named standard and level: **WCAG 2.2 Level AA**. State **fully**, **partially**, or **not yet** conformant — honestly.
3. **Known issues** — the parts that don't yet meet the standard, each citing the **WCAG success criterion** it fails and a target fix date, plus how to get an accessible alternative.
4. **Feedback & contact** — how to report a problem and request an alternative, with a response-time commitment.
5. **How it was prepared/tested** — date prepared/reviewed, and the method (self-assessment with axe + manual/AT, or a third-party audit and against what standard).
6. (For procurement) a **VPAT/ACR** table — see below.

### The three conformance claims

| Claim | Use when | Wording |
| --- | --- | --- |
| **Fully conformant** | The product meets WCAG 2.2 AA in full (needs an audit behind it). | "[Product] is fully conformant with WCAG 2.2 Level AA." |
| **Partially conformant** | Most of AA is met, with listed exceptions. | "[Product] is partially conformant with WCAG 2.2 Level AA — some content does not yet fully conform, listed below." |
| **Not yet conformant** | It does not meet most of AA. | "[Product] is not yet conformant with WCAG 2.2 Level AA. We are working towards it." |

Never claim "fully conformant" without a real audit — an unbacked claim is a legal and reputational risk.

---

## Fill-in-the-blanks template (voluntary statement)

Copy this and replace the `[bracketed]` text.

```markdown
# Accessibility statement for [product/site name]

[Organisation name] is committed to making [product name] accessible to as many people as possible, in line with the Web Content Accessibility Guidelines (WCAG) 2.2 Level AA.

This statement applies to [scope: e.g. the app at app.example.com].

## Using this product

We want everyone to be able to use [product name]. You should be able to:

- change colours, contrast, and font size using your browser or device settings
- zoom in up to 400% without content spilling off the screen
- navigate the product using a keyboard
- use it with a screen reader (recent versions of NVDA, JAWS, and VoiceOver)

We use the [shadcn/ui + Radix UI] component system, which provides an accessible baseline for interactive elements (keyboard, focus, and screen-reader support).

## How accessible this product is

[Choose ONE and adapt:]
[Product name] is partially conformant with WCAG 2.2 Level AA. We know some parts are not yet fully accessible:

- [e.g. some data charts do not yet have a text alternative — fails WCAG 1.1.1; fix planned by [date]]
- [e.g. colour contrast on [feature] falls below 4.5:1 in dark mode — fails WCAG 1.4.3; fix planned by [date]]

## Feedback and getting help

If you find an accessibility problem, or need information in a different format:

- email [email address]
- [add other contact details]

We aim to respond within [number] working days and will provide an accessible alternative where we can.

## How we tested this product

This statement was prepared on [date] and last reviewed on [date].

[Product name] was last tested on [date] by [your organisation / the third party that audited it]. We tested [scope] using [automated tools (axe) plus manual keyboard, zoom, and screen-reader testing / a formal WCAG 2.2 AA audit].
```

### Notes for developers filling this in

- Cite the **specific WCAG 2.2 success criterion** for each known issue, and give a target fix date.
- Keep it **honest and specific** — "we care about accessibility" with nothing behind it is worse than a candid partial claim.
- Review it at least annually and whenever the product changes materially.

---

## VPAT / ACR (for procurement)

A **VPAT** (Voluntary Product Accessibility Template) is the standard form; filled in, it produces an **Accessibility Conformance Report (ACR)**. Use the **VPAT 2.x WCAG edition** (or the "INT" edition to also cover EN 301 549 for the EU and Section 508 for the US).

For each WCAG 2.2 A/AA success criterion you record a **Conformance Level** and **Remarks**:

| Conformance level | Meaning |
| --- | --- |
| **Supports** | Meets the criterion. |
| **Partially Supports** | Some functionality does not meet it. |
| **Does Not Support** | Majority of functionality does not meet it. |
| **Not Applicable** | The criterion is not relevant to the product. |

Guidance:
- Base the ACR on **real testing** (automated + manual + AT), not a hopeful reading — buyers and their auditors check.
- Be precise in Remarks: name the affected component/screen and the nature of the gap.
- Keep the ACR dated and versioned; re-issue it when the product changes.
- The `/shadcn:audit` command and the `shadcn-accessibility-auditor` agent produce criterion-level findings you can map straight into VPAT rows.
