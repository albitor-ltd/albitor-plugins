Source: GOV.UK Design System (https://design-system.service.gov.uk/) and GOV.UK Frontend. Crown Copyright, licensed under the Open Government Licence v3.0. Distilled reference — verify against the live source for current detail.

# GOV.UK Frontend conventions: how to actually build it

GOV.UK Frontend (npm package **`govuk-frontend`**) provides the CSS, JavaScript, Nunjucks macros/templates and font/image assets needed to build pages that look and behave like GOV.UK. This file covers installing it, the two ways to consume components (Nunjucks vs HTML), the page template, initialising JS, progressive enhancement, the Prototype Kit vs production, Sass customisation, and consuming GOV.UK Frontend from a non-Nunjucks stack (React/Svelte).

## Installing

```bash
npm install govuk-frontend
```

Two consumption modes:

1. **Precompiled / "dist" files** — copy the compiled `govuk-frontend.min.css`, `govuk-frontend.min.js` and the `assets/` folder (fonts, images, favicons). Simplest; no build step. You cannot selectively include components or customise Sass settings.
2. **npm + your own build** — import the Sass and JS into your pipeline. Lets you selectively include component CSS/JS, use the Sass mixins/functions, override settings (colours, page width, font path), and use the Nunjucks template/macros.

Either way you must serve the **assets** (fonts and images). With a bundler, copy `node_modules/govuk-frontend/dist/govuk/assets` into your public assets path, or point the Sass `$govuk-assets-path` setting at where you serve them.

## Importing the styles (Sass)

Import everything:

```scss
@use "govuk-frontend/dist/govuk" as *;
```

(Older codebases use `@import "govuk-frontend/govuk/all";`.) To simplify paths, add `node_modules` to your Sass `loadPaths`/`includePaths` so you can `@use "govuk/all"`.

Selectively include components (only what you use) by importing the base plus individual component partials, e.g.:

```scss
@import "govuk-frontend/govuk/base";
@import "govuk-frontend/govuk/components/button/button";
```

### Sass settings / overrides
Override settings **before** importing the library by setting the `$govuk-*` variables, e.g.:

```scss
$govuk-page-width: 1100px;            // widen the container
$govuk-assets-path: "/assets/";       // where fonts/images are served
$govuk-global-styles: true;           // apply base styles to bare elements
$govuk-new-typography-scale: true;    // opt into the updated type scale
@use "govuk-frontend/dist/govuk" as *;
```

Use the provided functions/mixins (`govuk-colour()`, `govuk-spacing()`, `govuk-font()`, `govuk-responsive-margin()`) rather than hardcoding values, so your service tracks palette/scale changes.

### Off-GOV.UK setup
`govuk-template` and the components are fine to use off GOV.UK, but the GDS Transport typeface and the GOV.UK branding are not (licensing, not style — see the "is this service on GOV.UK?" table in `SKILL.md`). For an off-GOV.UK build, apply the Sass settings from `styles.md` (`$govuk-include-default-font-face: false` and a system `$govuk-font-family` such as `arial, helvetica, sans-serif`), do not deploy the `/assets/fonts/*` files, and replace the branded header/footer (crown logo, "GOV.UK" name, OGL, Crown copyright) with your own.

## Initialising JavaScript (`initAll`)

GOV.UK Frontend JS enhances components that carry a `data-module` attribute (e.g. `data-module="govuk-button"`). Import and run `initAll` once on page load:

```html
<script type="module">
  import { initAll } from "/assets/govuk-frontend.min.js";
  initAll();
</script>
```

(With a bundler: `import { initAll } from "govuk-frontend"; initAll();`.) You can also initialise a single component or only part of the page by `import`ing the component class and `new`-ing it against a scope element, instead of `initAll`. GOV.UK Frontend JS is shipped as ES modules.

## Progressive enhancement (non-negotiable)

The GOV.UK Service Manual mandates progressive enhancement: build with **HTML first**, then layer CSS, then JavaScript. JavaScript is an enhancement, never a requirement.

Why JS may not run: network drops, ad blockers/extensions, CDN/third-party outage, corporate firewalls rewriting content, mobile carriers transcoding, user settings, or a single JS error halting all subsequent scripts. JS is "not fault-tolerant" — one error stops the rest.

Mechanics in GOV.UK Frontend:
- The page template runs a tiny inline head script that adds the `govuk-frontend-supported` (and historically `js-enabled`) class to `<html>` only if the browser passes a feature check. CSS that styles JS-only behaviour is gated on this class, so non-JS users get the unenhanced (but working) version.
- Components degrade gracefully: Character count → plain textarea with the limit in hint text; Accordion/Tabs → all sections visible; Radios/Checkboxes conditional reveals → revealed content always reachable; Back link/Exit this page → real `href`s.

Practical rules:
- Make the service fully usable with HTML alone; only enhance equivalent HTML/CSS functionality with JS.
- Avoid single-page-app architectures that break browser navigation, the back button and accessibility. Server-render pages; use real links and form submits.
- Validate on the server (client-side validation is an enhancement only).
- Test with JS disabled and across assistive technologies.

## Components: Nunjucks macros vs plain HTML

**Nunjucks macros** are the canonical source. Import and call them with an options object:

```njk
{% from "govuk/components/button/macro.njk" import govukButton %}
{{ govukButton({ text: "Continue" }) }}

{% from "govuk/components/input/macro.njk" import govukInput %}
{{ govukInput({
  label: { text: "Full name", classes: "govuk-label--l", isPageHeading: true },
  id: "full-name",
  name: "fullName",
  autocomplete: "name"
}) }}
```

Macro names map to components as `govuk<Name>` (e.g. `govukButton`, `govukErrorSummary`, `govukDateInput`). Each component page on the Design System lists its macro options.

**Plain HTML** is equally valid: every component page provides copy-paste HTML with the right classes and ARIA. Use this when you're not on Nunjucks — but keep the classes, structure and ARIA exactly, and remember you must hand-maintain it when GOV.UK Frontend updates.

## Minimal working page template (plain HTML)

```html
<!DOCTYPE html>
<html lang="en" class="govuk-template">
  <head>
    <meta charset="utf-8">
    <title>Page title - Service name - GOV.UK</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="theme-color" content="#0b0c0c">
    <link rel="stylesheet" href="/assets/govuk-frontend.min.css">
    <link rel="icon" sizes="48x48" href="/assets/images/favicon.ico">
  </head>
  <body class="govuk-template__body">
    <script>
      document.body.parentNode.classList.add('govuk-frontend-supported');
    </script>

    <a href="#main-content" class="govuk-skip-link" data-module="govuk-skip-link">Skip to main content</a>

    <header class="govuk-header" data-module="govuk-header">
      <!-- govukHeader markup: crown logo + GOV.UK -->
    </header>

    <div class="govuk-width-container">
      <a href="/previous-page" class="govuk-back-link">Back</a>
      <main class="govuk-main-wrapper" id="main-content" role="main">
        <div class="govuk-grid-row">
          <div class="govuk-grid-column-two-thirds">
            <h1 class="govuk-heading-xl">Page heading</h1>
            <p class="govuk-body">Body content goes here.</p>
            <button type="submit" class="govuk-button" data-module="govuk-button">Continue</button>
          </div>
        </div>
      </main>
    </div>

    <footer class="govuk-footer">
      <!-- govukFooter markup: OGL + Crown copyright -->
    </footer>

    <script type="module">
      import { initAll } from "/assets/govuk-frontend.min.js";
      initAll();
    </script>
  </body>
</html>
```

In Nunjucks this is far shorter — extend the supplied template and fill blocks:

```njk
{% extends "govuk/template.njk" %}
{% from "govuk/components/button/macro.njk" import govukButton %}

{% block pageTitle %}Page title - Service name - GOV.UK{% endblock %}

{% block content %}
  <div class="govuk-grid-row">
    <div class="govuk-grid-column-two-thirds">
      <h1 class="govuk-heading-xl">Page heading</h1>
      {{ govukButton({ text: "Continue" }) }}
    </div>
  </div>
{% endblock %}
```

## Prototype Kit vs production

- **GOV.UK Prototype Kit** — a Node/Express + Nunjucks tool for quickly building realistic, clickable prototypes to test with users. It bundles GOV.UK Frontend and adds prototyping conveniences (routing, session data, plugins). It is for research/testing only — not production. Don't ship prototype-kit code as a live service; prototypes deliberately cut corners (no real backend, relaxed validation/security).
- **Production** — install `govuk-frontend` directly into your real application, build assets properly, validate server-side, meet accessibility and security requirements, and only include the components/JS you use.

## Coding in the open

GOV.UK services follow the Service Standard's "make all new source code open" point: develop in public repositories under an open licence (e.g. MIT for code), keeping secrets/credentials and personal data out of the repo. GOV.UK Frontend and the Design System are themselves open source (alphagov on GitHub) under the MIT licence, with content under the Open Government Licence v3.0.

## Consuming GOV.UK Frontend from React / Svelte / other non-Nunjucks stacks

GOV.UK Frontend's contract is its **HTML structure + classes + ARIA + behaviour**, not Nunjucks. To use it from React/Svelte/Vue/etc:

1. **Copy the component HTML** from the Design System component page (or render the Nunjucks macro once and copy its output) into your framework component. Keep the exact class names, element structure, `id`/`name`/`for` wiring, `aria-*` attributes and `data-module` values — these are the contract.
2. **Keep the GOV.UK CSS**: import the GOV.UK Frontend Sass/CSS into your build so the copied classes resolve. Serve the fonts/images assets.
3. **Port the JS behaviour**: GOV.UK's JS finds elements by `data-module` and enhances them. Two options:
   - Re-use GOV.UK Frontend's own JS by importing the component class and instantiating it against your rendered DOM node (e.g. in a React `useEffect`/Svelte `onMount`, `new Button(node)` or `initAll()` scoped to a ref). Tear it down on unmount where needed.
   - Or re-implement the same behaviour natively in your framework, preserving the same DOM changes, ARIA states and keyboard interactions.
4. **Preserve progressive enhancement**: prefer server-side rendering (Next.js, SvelteKit SSR, etc.) so the HTML works before hydration; avoid client-only SPAs for transactional government services. Real `<form>` submits and `<a href>` links, not JS-only handlers.
5. Watch for the `govuk-frontend-supported` class gate — include the head script so JS-enhanced styling activates, and make sure framework hydration doesn't strip GOV.UK's DOM enhancements.

Caveat: there is no official React/Svelte component library; community ports exist but may lag GOV.UK Frontend. Treat the live Design System HTML as the source of truth and keep your ports in sync when GOV.UK Frontend releases updates (including the 2025 brand refresh).
