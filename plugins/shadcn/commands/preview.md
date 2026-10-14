---
description: Open or serve the shadcn/ui component preview page (the kitchen-sink showcase).
argument-hint: "[leave blank to open, or 'serve' to run a local static server]"
---

Show the user the bundled shadcn/ui **preview page** — a self-contained, build-free kitchen-sink that renders the core shadcn/ui components (buttons and their variants, inputs, form fields, card, dialog, tabs, table, badges, alert, checkbox, switch, select, tooltip, and more) in light and dark themes. It is the "what does shadcn/ui look like?" reference consumed by the describe-and-build flow, the design review baseline, and the handover pack. Context: **$ARGUMENTS**

The page lives at `${CLAUDE_PLUGIN_ROOT}/preview/index.html` and is fully standalone — it inlines its CSS/JS and needs no build step, network, or server.

Do the following:

1. Resolve the absolute path to `preview/index.html` inside this plugin (`${CLAUDE_PLUGIN_ROOT}/preview/index.html`).
2. Open it for the user:
   - If they just want to look, it opens straight from `file://` — give them the path and, on macOS, offer to run `open "<path>"`; on Linux `xdg-open "<path>"`.
   - If they asked to **serve** it (or want a URL, e.g. to view in a remote/container browser), start a static server in the `preview/` directory, e.g. `python3 -m http.server 8000` (or `npx serve`), and give them `http://localhost:8000/`.
3. Explain what the page is for: it is the visual **baseline** for the shadcn/ui look — use it to sanity-check that a build's UI matches the design system, as the reference image in a design review, and as an artefact in the handover pack. It is a static showcase, not a live component library — for real usage, scaffold components with `/shadcn:component`.
4. If the user wants a preview of *their own* app's screens rather than the reference kitchen-sink, point them at running their app's dev server instead; this page is the design-system reference, not their app.
