---
description: Open or serve the crownless GOV.UK Design System preview page (kitchen-sink component showcase).
argument-hint: "[optional: 'open' to launch a browser, or 'serve' to start a local server]"
---

The user wants to view the plugin's GOV.UK Design System preview page: **$ARGUMENTS**

The preview lives at `${CLAUDE_PLUGIN_ROOT}/preview/index.html`. It is a self-contained, static kitchen-sink showcase of the Design System components, patterns and styles — **crownless**: no crown logo, no "GOV.UK" logotype, no GDS Transport font, and no Open Government Licence / Crown-copyright statements, matching how these standards are used for a **non-government** service. It opens straight from `file://` with no build step.

Do the following:

1. Resolve the absolute path to `${CLAUDE_PLUGIN_ROOT}/preview/index.html` and confirm it exists.
2. Open or serve it, picking the mode from the argument (default: **open**):
   - **open** — open the file directly in the default browser:
     - macOS: `open "${CLAUDE_PLUGIN_ROOT}/preview/index.html"`
     - Linux: `xdg-open "${CLAUDE_PLUGIN_ROOT}/preview/index.html"`
     - Windows: `start "" "%CLAUDE_PLUGIN_ROOT%\preview\index.html"`
   - **serve** — serve the `preview/` directory over HTTP and print the URL (useful in a sandbox or over SSH):
     `python3 -m http.server 8080 --directory "${CLAUDE_PLUGIN_ROOT}/preview"` then visit <http://localhost:8080/>.
3. Tell the user what the page demonstrates: buttons, form fields (inputs, textarea, select, radios, checkboxes with conditional reveal, date input), error summary and inline errors, notification banners, tags, panel, tables, tabs, summary list, task list, accordion, breadcrumbs, phase banner, and the standard black header bar **without** the crown.
4. Point out that this is the **preview-page capability** every Albitor design-system plugin ships: it is the baseline for the create / describe-and-build flow, the post-build design review, and the handover pack.

If asked to adapt it (rename the service, drop a component, restyle), copy `preview/index.html` into the target project and edit there — treat the plugin's copy as the reference original.
