---
description: Review frontend changes against both shadcn/ui + Radix conformance and WCAG 2.2 AA.
argument-hint: "[file/dir path, or leave blank for the current changes]"
---

Review the frontend in scope: **$ARGUMENTS** (if empty, review the current uncommitted changes; fall back to recently edited frontend files).

Load **both** skills: `shadcn-design-system` and `shadcn-accessibility`. Then review on two axes:

1. **Design System conformance** (`references/components.md`, `patterns.md`, `styles.md`):
   - Is a shadcn/Radix component used where one exists, instead of a hand-rolled element? Flag every **ad-hoc component** — a raw `<button>`, bare `<input>`/`<select>`/`<textarea>`, a custom modal `<div>`, a bespoke dropdown/tooltip/tabs — that re-implements something already in `components/ui/`. This is the "0 ad-hoc components" bar: the UI should be built from shadcn/Radix primitives, not re-implementations of them.
   - Are components used with their Radix behaviour intact (not `outline-none` without a `focus-visible` replacement, `asChild` not dropping semantics, `data-state`/ARIA not overridden)?
   - Is theming via the Tailwind tokens / CSS variables (`bg-primary`, `text-foreground`, `border-input`, `ring-ring`) rather than hardcoded hex or arbitrary values? Are variants expressed through the component's props/`cva` variants rather than ad-hoc class soup?
   - Are classes merged with `cn()` (clsx + tailwind-merge) so overrides win predictably?

2. **Accessibility** (`references/wcag-2.2.md`): run the WCAG 2.2 AA checks — alt text, `Label`/input association (or shadcn `Form`), focus visibility, heading order, contrast, `lang`, announced + linked errors, target size, keyboard operability, `prefers-reduced-motion`.

Report findings grouped by axis, each with `file:line`, severity (Blocker/Major/Minor), the problem, and a concrete fix (name the shadcn/Radix component or token to use). Note any issue that needs a real-browser or assistive-tech check rather than static review. Finish with a prioritised to-do list. Be specific and code-level; don't restate the standards in the abstract.
