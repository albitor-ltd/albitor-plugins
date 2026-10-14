---
description: Look up or scaffold a shadcn/ui component (CLI add command, usage, Radix primitive, accessibility notes).
argument-hint: "[component name or what you're trying to build]"
---

The user wants help with a shadcn/ui component: **$ARGUMENTS**

Do the following:

1. Load the `shadcn-design-system` skill and read `references/components.md` (and `references/patterns.md` if the request describes a flow rather than a single element).
2. Identify the right component(s). If the request is vague (e.g. "let users pick a date"), recommend the correct component and say why (e.g. `Calendar` + `Popover` = Date Picker), and mention any element they should *not* hand-roll instead.
3. Provide:
   - The **install command** — `npx shadcn@latest add <component>` (list every component the snippet needs), **and**
   - Realistic **usage code** (React/TSX) importing from the project's alias (e.g. `@/components/ui/<component>`), with the right props/composition.
4. Call out the **accessibility requirements** for that component: which **Radix primitive** it wraps and what that gives for free (focus management, keyboard, ARIA `role`/`aria-*`, `data-state`), plus what the author still owns — an associated `Label` (`htmlFor`/`id`) for inputs, an accessible name for icon-only buttons (`aria-label` or `sr-only` text), a `DialogTitle`/`DialogDescription` for dialogs, colour contrast, and "when not to use".
5. If the component needs client interactivity, note that it is a Client Component (`"use client"` in the RSC/App Router world) and that Radix handles the keyboard/focus behaviour — do not re-implement it.
6. Match the surrounding codebase: detect whether the project is React (Next.js/Vite), or a Svelte/Vue port, and tailor the snippet. Never hardcode hex colours — use the Tailwind theme tokens (`bg-primary`, `text-muted-foreground`, `border-input`, `ring-ring`, etc.) backed by the CSS variables, so theming and dark mode keep working. Merge classes with the `cn()` helper, don't concatenate strings.

If no argument was given, ask what they're building and list the component categories from the reference.
