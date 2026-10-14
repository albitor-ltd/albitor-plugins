# Accessibility statements (consumer / general-purpose products)

> **Sources:** [W3C WAI accessibility-statement generator and guidance](https://www.w3.org/WAI/planning/statements/), plus the conformance expectations of [EN 301 549](https://www.etsi.org/standards) / the European Accessibility Act, Section 508, and the ADA.
> Distilled reference — verify the exact obligations for your market against the live sources. This template is for **consumer and general-purpose** products; a public sector body with a statutory template must use that jurisdiction's mandated wording instead.

## When to publish one

An accessibility statement is **good practice for any product** and is **expected or required** where a regime applies:

- **European Accessibility Act (EAA) / EN 301 549** — many products and services sold in the EU (e-commerce, banking, transport, e-books, consumer devices) must be accessible from 28 June 2025 and should document conformance.
- **US ADA / Section 508** — private-sector web accessibility is enforced through the ADA; federal and federally-funded bodies use Section 508 (and often publish a VPAT/ACR).
- Even with no legal trigger, a clear statement builds trust and gives users a route to report problems.

Publish it as an **HTML page** linked from a persistent location (typically the footer), review it **regularly** (at least annually) and update it whenever the product changes.

## What a good statement contains

1. **Commitment** — a short statement that you aim to make the product usable by as many people as possible.
2. **Conformance status** — against **WCAG 2.2 AA**: fully, partially, or not yet conformant (be honest — most products are "partially"). Tie any claim to the standard and how it was assessed.
3. **What people can do** — the accessibility features that work (keyboard, zoom to 400%, screen-reader support, contrast, etc.).
4. **Known limitations** — the areas that don't yet meet WCAG 2.2 AA, the criteria they fail, and target fix dates. Honesty here is what makes the statement credible.
5. **Feedback and contact** — how to report a problem or request an alternative, with a response-time commitment.
6. **How it was tested** — self-assessment (axe + manual + AT) or an external audit, against what standard, and on what date.

### Conformance status options

| Option | Use when | Wording |
| --- | --- | --- |
| **Fully conformant** | The product meets WCAG 2.2 AA in full (audit-backed). | "This product is fully conformant with WCAG 2.2 level AA." |
| **Partially conformant** | Most of WCAG 2.2 AA is met, with listed exceptions. | "This product is partially conformant with WCAG 2.2 level AA — it conforms to most of the standard, with the exceptions listed below." |
| **Not yet conformant** | The product does not yet meet most of WCAG 2.2 AA. | "This product is not yet conformant with WCAG 2.2 level AA. We are working towards it." |

---

## Fill-in-the-blanks template

Copy this and replace the `[bracketed]` text.

```markdown
# Accessibility statement for [product / website name]

[Organisation name] is committed to making [product name] accessible to as many
people as possible, in line with the Web Content Accessibility Guidelines
(WCAG) 2.2 level AA.

This statement applies to [scope: e.g. the app at app.example.com].

## How accessible this product is

We want everyone to be able to use [product name]. You should be able to:

- navigate the whole product with a keyboard
- zoom in up to 400% without content spilling off the screen or being cut off
- use it with a screen reader (recent versions of NVDA, JAWS and VoiceOver)
- change colours and contrast using your browser or device settings

## Conformance status

[Choose ONE and adapt:]
This product is partially conformant with WCAG 2.2 level AA. "Partially
conformant" means some parts do not yet fully meet the standard — these are
listed below.

## Known accessibility issues

We are aware of the following issues and are working to fix them:

- [Describe the issue and the WCAG 2.2 criterion it fails, e.g.] Some data
  visualisations do not yet have a text alternative, so information is not
  available to screen-reader users. This fails WCAG 2.2 success criterion 1.1.1
  (Non-text Content). We plan to add data tables by [date].
- [Add others...]

## Feedback and reporting problems

If you find an accessibility problem, or need information in a different format:

- email [email address]
- [add other contact details]

We aim to respond within [number] working days.

## How we tested this product

This statement was prepared on [date] and last reviewed on [date].

[product name] was last tested on [date]. We tested [scope] using [automated
testing with axe, manual keyboard and screen-reader testing / an external audit
by [auditor]] against WCAG 2.2 level AA.
```

### Notes for developers filling this in

- Every limitation you list should cite the **WCAG 2.2 success criterion** it fails and, ideally, a target fix date.
- Don't claim "fully conformant" without an audit — prefer "tested against WCAG 2.2 AA using…".
- Keep it honest and specific; a statement that hides known issues undermines trust and, under some regimes, is itself a compliance problem.
- If a regulator (EAA, ADA, Section 508) applies to you, check whether a specific format (e.g. a VPAT / Accessibility Conformance Report) is expected in addition to this statement.
