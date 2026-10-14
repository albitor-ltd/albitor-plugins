---
description: Look up or scaffold a GOV.UK Design System component (markup, macro, accessibility notes).
argument-hint: "[component name or what you're trying to build]"
---

The user wants help with a GOV.UK Design System component: **$ARGUMENTS**

Do the following:

1. Load the `govuk-design-system` skill and read `references/components.md` (and `references/patterns.md` if the request describes a flow rather than a single element).
2. Identify the right component(s). If the request is vague (e.g. "let users pick a date"), recommend the correct component and say why; mention any component they should *not* use.
3. Provide:
   - The **Nunjucks macro** form (`govuk*`) with realistic options, **and**
   - The equivalent **plain HTML** with all `govuk-*` classes, for non-Nunjucks stacks.
4. Call out the **accessibility requirements** for that component (labels/legends, error handling, focus, `data-module` + `initAll` if it needs JS) and any "when not to use" guidance.
5. If the component needs JavaScript, show how it is initialised and stress that the non-JS HTML must still work (progressive enhancement).
6. Match the surrounding codebase: detect whether the project uses Nunjucks, React, Svelte, or plain HTML and tailor the snippet to it. Do not hardcode hex colours — use Design System classes / `govuk-colour()`.

If no argument was given, ask what they're building and list the component categories from the reference.
