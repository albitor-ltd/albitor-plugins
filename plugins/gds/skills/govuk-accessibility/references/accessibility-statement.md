# Accessibility statements

> **Sources:** [GOV.UK: Make your website or app accessible and publish an accessibility statement](https://www.gov.uk/guidance/make-your-website-or-app-accessible-and-publish-an-accessibility-statement), [GOV.UK model accessibility statement](https://www.gov.uk/guidance/model-accessibility-statement), and the [sample accessibility statement](https://www.gov.uk/government/publications/sample-accessibility-statement).
> Crown Copyright / W3C. GOV.UK guidance licensed under the Open Government Licence v3.0; WCAG 2.2 © W3C. Distilled reference — verify against the live source for current detail.

## When a statement is required

Every public sector website and mobile app in scope of the regulations **must** publish an accessibility statement. It is a **legal requirement** — both the statement's existence and parts of its wording are mandated.

- **Websites:** publish as an **HTML page**, linked prominently from **every page** (typically the footer).
- **Mobile apps:** make available on the app's download page (app store), on a website, or both — in an accessible format.
- Review it **regularly** (at least annually) and update it whenever the service changes.

## Required content

A compliant statement must include:

1. **A compliance status statement** — choose one of three options (below), citing **WCAG 2.2 AA**.
2. **Non-accessible content** — if not fully compliant, list the parts that don't meet the standard, **grouped by reason**:
   - non-compliance with the regulations;
   - content exempt as a **disproportionate burden**;
   - content **outside the scope** of the regulations (e.g. pre-2018 documents, third-party content).
   Explain how people can **get an alternative** for inaccessible content.
3. **Feedback and contact information** — how to report problems and request accessible alternatives, with a response commitment.
4. **Enforcement procedure** — the legally-required EHRC/EASS (or ECNI) wording (below).
5. **Preparation of this statement** — when it was prepared/last reviewed and **how** it was tested (self-assessment or third-party audit, and against what standard).

### The three compliance status options

| Option | Use when | Wording |
| --- | --- | --- |
| **Fully compliant** | The service meets WCAG 2.2 AA in full. | "This website is fully compliant with the Web Content Accessibility Guidelines version 2.2 AA standard." |
| **Partially compliant** | Most of WCAG 2.2 AA is met, with listed exceptions/exemptions. | "This website is partially compliant with the Web Content Accessibility Guidelines version 2.2 AA standard, due to the non-compliances and exemptions listed below." |
| **Not compliant** | The service does not meet most of WCAG 2.2 AA. | "This website is not compliant with the Web Content Accessibility Guidelines version 2.2 AA standard." |

---

## Fill-in-the-blanks template (GDS model statement)

Copy this and replace the `[bracketed]` text. Keep the **Enforcement procedure** wording verbatim — it is legally required.

```markdown
# Accessibility statement for [website/app name]

This accessibility statement applies to [scope: e.g. the www.example.org.uk website / the Example mobile app].

This website is run by [organisation name]. We want as many people as possible to be able to use this website. For example, that means you should be able to:

- change colours, contrast levels and fonts using browser or device settings
- zoom in up to 400% without the text spilling off the screen
- navigate most of the website using a keyboard or speech recognition software
- listen to most of the website using a screen reader (including the most recent versions of JAWS, NVDA and VoiceOver)

We’ve also made the website text as simple as possible to understand.

[AbilityNet](https://mcmw.abilitynet.org.uk/) has advice on making your device easier to use if you have a disability.

## How accessible this website is

[Choose ONE of the following and adapt:]
[We know some parts of this website are not fully accessible: list the main issues in plain English, e.g.]
- some older PDF documents are not fully accessible to screen reader software
- live video streams do not have captions

## Feedback and contact information

If you need information on this website in a different format like accessible PDF, large print, easy read, audio recording or braille:

- email [email address]
- call [phone number]
- [add other contact details, e.g. postal address]

We’ll consider your request and get back to you in [number] days.

[If you cannot view the map on our ‘contact us’ page, call or email us for directions.]

## Reporting accessibility problems with this website

We’re always looking to improve the accessibility of this website. If you find any problems not listed on this page or think we’re not meeting accessibility requirements, contact: [name and/or email address and phone number of someone in your organisation].

## Enforcement procedure

The Equality and Human Rights Commission (EHRC) is responsible for enforcing the Public Sector Bodies (Websites and Mobile Applications) (No. 2) Accessibility Regulations 2018 (the ‘accessibility regulations’). If you’re not happy with how we respond to your complaint, contact the Equality Advisory and Support Service (EASS).

[For organisations based in Northern Ireland, replace the second sentence with:]
[If you’re not happy with how we respond to your complaint, contact the Equality Commission for Northern Ireland (ECNI).]

## Technical information about this website’s accessibility

[Organisation name] is committed to making its website accessible, in accordance with the Public Sector Bodies (Websites and Mobile Applications) (No. 2) Accessibility Regulations 2018.

### Compliance status

[Choose ONE:]
This website is fully compliant with the Web Content Accessibility Guidelines version 2.2 AA standard.

This website is partially compliant with the Web Content Accessibility Guidelines version 2.2 AA standard, due to the non-compliances and exemptions listed below.

This website is not compliant with the Web Content Accessibility Guidelines version 2.2 AA standard.

## Non-accessible content

The content listed below is non-accessible for the following reasons.

### Non-compliance with the accessibility regulations

[List accessibility problems and the WCAG 2.2 AA success criteria they fail, e.g.:]
- Some images do not have a text alternative, so people using a screen reader cannot access the information. This fails WCAG 2.2 success criterion 1.1.1 (non-text content). We plan to add text alternatives by [date].

### Disproportionate burden

[List items you’ve assessed as a disproportionate burden to fix, with a reference to your assessment. Remove this heading if not applicable.]

### Content that’s not within the scope of the accessibility regulations

[List exempt content, e.g. pre-23 September 2018 PDFs, third-party content, live video, online maps. Remove this heading if not applicable.]

## Preparation of this accessibility statement

This statement was prepared on [date]. It was last reviewed on [date].

This website was last tested on [date]. The test was carried out by [your organisation / the name of the third party that did the audit].

[Describe how you tested, e.g.:] We tested [scope of test] using [automated tools plus manual testing / a sample of pages / etc.]. [If audited:] A full accessibility audit was carried out by [auditor name].
```

### Notes for developers filling this in

- The **Enforcement procedure** paragraph is fixed legal wording — do not paraphrase it.
- Every non-compliance you list should cite the **WCAG 2.2 success criterion** it fails and, ideally, a target fix date.
- "Disproportionate burden" must be backed by a **documented assessment**; don't list items there to avoid work.
- Keep the statement honest and specific — GDS monitoring checks statements against the actual service.
